/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mtls

import (
	"reflect"
	"sync"
	"sync/atomic"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

const systemValidation = "system"

var (
	secretManagerInstance = &secretManager{
		validations: make(map[string]*validation),
	}
	// sdsCallbacks is used to sense that an asynchronous certificate configuration is successfully generated or updated
	sdsCallbacks = map[string]func(*v2.TLSConfig){}
)

func RegisterSdsCallback(name string, f func(*v2.TLSConfig)) {
	sdsCallbacks[name] = f
}

func addOrUpdateProvider(index string, cfg *v2.TLSConfig) *sdsProvider {
	return secretManagerInstance.AddOrUpdateProvider(index, cfg)
}

func ClearSecretManager() {
	secretManagerInstance.mutex.Lock()
	defer secretManagerInstance.mutex.Unlock()
	secretManagerInstance.validations = make(map[string]*validation)
	log.DefaultLogger.Infof("[mtls] [sds provider] clear all providers")
}

// secretManager
type secretManager struct {
	mutex       sync.Mutex
	validations map[string]*validation
}

func (mng *secretManager) addOrUpdatePemProvider(sdsConfig *v2.SdsConfig) *pemProvider {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()
	// found validation by sds config
	validationName := systemValidation
	if sdsConfig.ValidationConfig != nil {
		validationName = sdsConfig.ValidationConfig.Name
	}
	v, ok := mng.validations[validationName]
	if !ok {
		// add a validation
		v = &validation{
			// only system validation allows empty
			expectedEmpty: validationName == systemValidation,
			certificates:  make(map[string]*pemProvider),
		}
		mng.validations[validationName] = v
	}
	// found pem provider by sds config
	certName := sdsConfig.CertificateConfig.Name
	p, ok := v.certificates[certName]
	if !ok {
		// add a new pem provider
		p = &pemProvider{
			sdsProviders:  make(map[string]*sdsProvider),
			sdsConfig:     sdsConfig,
			expectedEmpty: v.expectedEmpty,
			rootca:        v.rootca,
		}
		v.certificates[certName] = p
		// set a certificate callback
		client := GetSdsClient(p.sdsConfig.CertificateConfig.SdsConfig)
		client.AddUpdateCallback(p.sdsConfig.CertificateConfig.Name, p.setCertificate)
		if !p.expectedEmpty {
			client.AddUpdateCallback(p.sdsConfig.ValidationConfig.Name, mng.setValidation)
		}
		log.DefaultLogger.Infof("[mtls] [sds provider] add a new pem provider: %s.%s", validationName, certName)
	} else {
		// update sds config
		if !reflect.DeepEqual(p.sdsConfig, sdsConfig) {
			p.sdsConfig = sdsConfig
			_ = GetSdsClient(p.sdsConfig.CertificateConfig.SdsConfig)
		}
	}
	return p
}

// AddOrUpdateProvider will create a new sds provider or update an exists sds provider by the index string.
func (mng *secretManager) AddOrUpdateProvider(index string, cfg *v2.TLSConfig) *sdsProvider {
	p := mng.addOrUpdatePemProvider(cfg.SdsConfig)
	return p.addOrUpdateSdsProvider(index, cfg)
}

// setValidation is called in sds client
func (mng *secretManager) setValidation(name string, secret *types.SdsSecret) {
	mng.mutex.Lock()
	defer mng.mutex.Unlock()
	v, ok := mng.validations[name]
	if !ok {
		return
	}
	if secret.ValidationPEM != "" {
		v.rootca = secret.ValidationPEM
		log.DefaultLogger.Infof("[mtls] [sds provider] provider %s receive a validation set", name)
		// set the validation
		for _, cert := range v.certificates {
			cert.setValidation(v.rootca)
		}
	}
}

type validation struct {
	// if expectedEmpty is true, we expected the rootca is empty string
	// that means we use the system rootca.
	expectedEmpty bool
	// rootca stored the validation pem string
	rootca string
	// certificates stored the certificates that are signed by the validation
	certificates map[string]*pemProvider
}

type pemProvider struct {
	mutex sync.Mutex
	// expectedEmpty and rootca is from validation
	expectedEmpty bool
	rootca        string
	// cert stored the certificate pem string
	cert string
	// key stored the private key pem string
	key string
	// sds config
	sdsConfig *v2.SdsConfig
	// sdsProviders stored the tls configs that are created by the certificate and key.
	sdsProviders map[string]*sdsProvider
}

func (pp *pemProvider) addOrUpdateSdsProvider(index string, cfg *v2.TLSConfig) *sdsProvider {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()
	// add or update sds provider by index, and return it.
	p, ok := pp.sdsProviders[index]
	if !ok {
		p = &sdsProvider{
			info: &SecretInfo{
				Certificate:  pp.cert,
				PrivateKey:   pp.key,
				Validation:   pp.rootca,
				NoValidation: pp.expectedEmpty,
			},
		}
		pp.sdsProviders[index] = p
	}
	// update sds provider
	p.updateConfig(cfg)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[mtls] [sds provider] add or update sds provider %s", index)
	}
	return p
}

func (pp *pemProvider) setCertificate(name string, secret *types.SdsSecret) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()

	if secret.CertificatePEM != "" {
		pp.cert = secret.CertificatePEM
		pp.key = secret.PrivateKeyPEM

	}

	for index, p := range pp.sdsProviders {
		p.setCertificate(pp.cert, pp.key)
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("[mtls] [pem provider] provider %s receive a cerificate set, set to %s", name, index)
		}
	}
}

func (pp *pemProvider) setValidation(rootca string) {
	pp.mutex.Lock()
	defer pp.mutex.Unlock()

	pp.rootca = rootca

	for _, p := range pp.sdsProviders {
		p.setValidation(rootca)
	}
}

// sdsProvider is an implementation of types.Provider
// sdsProvider stored a tls context that makes by sds
type sdsProvider struct {
	value  atomic.Value // store tlsContext
	config atomic.Value // store *v2.TLSConfig
	info   *SecretInfo
}

func (p *sdsProvider) updateConfig(cfg *v2.TLSConfig) {
	p.config.Store(cfg)
	p.update()
}

func (p *sdsProvider) setValidation(v string) {
	p.info.Validation = v
	p.update()
}

func (p *sdsProvider) setCertificate(cert, key string) {
	p.info.Certificate = cert
	p.info.PrivateKey = key
	p.update()
}

func (p *sdsProvider) update() {
	if !p.info.Full() {
		return
	}
	v := p.config.Load()
	cfg, _ := v.(*v2.TLSConfig)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[mtls] [sds provider] sds provider update a tls context by config: %v", cfg)
	}
	ctx, err := newTLSContext(cfg, p.info)
	if err != nil {
		log.DefaultLogger.Alertf("sds.update", "[mtls] [sds] update tls context failed: %v", err)
		return
	}
	p.value.Store(ctx)
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("[mtls] [sds] update tls context success")
	}
	// notify certificates updates
	for _, cb := range cfg.Callbacks {
		if f, ok := sdsCallbacks[cb]; ok {
			f(cfg)
		}
	}
}

func (p *sdsProvider) GetTLSConfigContext(client bool) *types.TLSConfigContext {
	v := p.value.Load()
	ctx, ok := v.(*tlsContext)
	if !ok {
		return nil
	}
	return ctx.GetTLSConfigContext(client)
}

func (p *sdsProvider) MatchedServerName(sn string) bool {
	v := p.value.Load()
	ctx, ok := v.(*tlsContext)
	if !ok {
		return false
	}
	return ctx.MatchedServerName(sn)
}

func (p *sdsProvider) MatchedALPN(protos []string) bool {
	v := p.value.Load()
	ctx, ok := v.(*tlsContext)
	if !ok {
		return false
	}
	return ctx.MatchedALPN(protos)
}

func (p *sdsProvider) Ready() bool {
	v := p.value.Load()
	_, ok := v.(*tlsContext)
	return ok
}

func (p *sdsProvider) Empty() bool {
	return false
}
