package xds

import (
	"github.com/juju/errors"
	"sync"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"

	"sofastack.io/sofa-mosn/pkg/types"

	"sofastack.io/sofa-mosn/pkg/log"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2/auth"
	v2 "sofastack.io/sofa-mosn/pkg/xds/v2"
)

type SdsClientImpl struct {
	SdsConfigMap   map[string]*auth.SdsSecretConfig
	SdsCallbackMap map[string]v2.SdsUpdateCallbackFunc
	updatedLock    sync.Mutex
	sdsSubscriber  *v2.SdsSubscriber
}

var sdsClient *SdsClientImpl
var sdsClientLock sync.Mutex

// GetSdsClientImpl use by tls module , when get sds config from xds
func GetSdsClient(config *auth.SdsSecretConfig) v2.SdsClient {
	sdsClientLock.Lock()
	defer sdsClientLock.Unlock()
	if sdsClient != nil {
		return sdsClient
	} else {
		sdsClient = &SdsClientImpl{
			SdsConfigMap:   make(map[string]*auth.SdsSecretConfig),
			SdsCallbackMap: make(map[string]v2.SdsUpdateCallbackFunc),
		}
		// For Istio , sds config should be the same
		// So we use first sds config to init sds subscriber
		sdsClient.sdsSubscriber = v2.NewSdsSubscriber(sdsClient, config.SdsConfig, ServiceNode, ServiceCluster)
		err := sdsClient.sdsSubscriber.Start()
		if err != nil {
			log.DefaultLogger.Errorf("[sds] [sdsclient] sds subscriber start fail", err)
			return nil
		}
		return sdsClient
	}
	return nil
}

// CloseSdsClientImpl used only mosn exit
func CloseSdsClient() {
	sdsClientLock.Lock()
	defer sdsClientLock.Unlock()
	if sdsClient != nil && sdsClient.sdsSubscriber != nil {
		sdsClient.sdsSubscriber.Stop()
		sdsClient.sdsSubscriber = nil
		sdsClient = nil
	}
}

func (client *SdsClientImpl) AddUpdateCallback(sdsConfig *auth.SdsSecretConfig, callback v2.SdsUpdateCallbackFunc) error {
	if sdsClient == nil {
		return errors.New(" sds client not init!")
	}
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	client.SdsConfigMap[sdsConfig.Name] = sdsConfig
	client.SdsCallbackMap[sdsConfig.Name] = callback
	client.sdsSubscriber.SendSdsRequest(sdsConfig.Name)
	return nil
}

// DeleteUpdateCallback
func (client *SdsClientImpl) DeleteUpdateCallback(sdsConfig *auth.SdsSecretConfig) error {
	if sdsClient == nil {
		return errors.New(" sds client not init!")
	}
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	delete(client.SdsConfigMap, sdsConfig.Name)
	delete(client.SdsCallbackMap, sdsConfig.Name)
	return nil
}

// SetSecret invoked when sds subscriber get secret response
func (client *SdsClientImpl) SetSecret(name string, secret *auth.Secret) {
	client.updatedLock.Lock()
	defer client.updatedLock.Unlock()
	if fc, ok := client.SdsCallbackMap[name]; ok {
		log.DefaultLogger.Debugf("[xds] [sds client],set secret = %v", name)
		mosnSecret := &types.SDSSecret{
			Name: secret.Name,
		}
		if validateSecret, ok := secret.Type.(*auth.Secret_ValidationContext); ok {
			ds := validateSecret.ValidationContext.TrustedCa.Specifier.(*core.DataSource_InlineBytes)
			mosnSecret.ValidationPEM = string(ds.InlineBytes)
		}
		if tlsCert, ok := secret.Type.(*auth.Secret_TlsCertificate); ok {
			certSpec, _ := tlsCert.TlsCertificate.CertificateChain.Specifier.(*core.DataSource_InlineBytes)
			priKey, _ := tlsCert.TlsCertificate.PrivateKey.Specifier.(*core.DataSource_InlineBytes)
			mosnSecret.CertificatePEM = string(certSpec.InlineBytes)
			mosnSecret.PrivateKeyPEM = string(priKey.InlineBytes)
		}
		fc(name, mosnSecret)
	}
}
