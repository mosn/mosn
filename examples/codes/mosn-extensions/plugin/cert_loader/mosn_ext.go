package main

import (
	"crypto/x509"
	"encoding/json"
	"fmt"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
	"mosn.io/mosn/pkg/plugin"
	"mosn.io/mosn/pkg/plugin/proto"
	"mosn.io/mosn/pkg/types"
)

func init() {
	mtls.Register("loader", &factory{})
}

type factory struct {
}

func (f *factory) CreateConfigHooks(config map[string]interface{}) mtls.ConfigHooks {
	return newHooks()
}

type hooks struct {
	client *plugin.Client
}

func newHooks() *hooks {
	client, err := plugin.Register("loader", nil)
	if err != nil {
		log.DefaultLogger.Fatalf("register loader failed: %v", err)
	}
	return &hooks{client}
}

type certInfo struct {
	CertPem string `json:"cert_pem"`
	KeyPem  string `json:"key_pem"`
	CAPem   string `json:"ca_pem"`
}

func (h *hooks) getInfoFromLoader() (*certInfo, error) {
	resp, err := h.client.Call(&proto.Request{}, time.Second)
	if err != nil {
		return nil, err
	}
	body := resp.Body
	info := &certInfo{}
	if err := json.Unmarshal(body, info); err != nil {
		return nil, fmt.Errorf("get info from loader error: %v", err)
	}
	return info, nil
}

func (h *hooks) GetCertificate(certIndex, keyIndex string) (tls.Certificate, error) {
	info, err := h.getInfoFromLoader()
	if err != nil {
		return tls.Certificate{}, err
	}
	return tls.X509KeyPair([]byte(info.CertPem), []byte(info.KeyPem))
}

func (h *hooks) GetX509Pool(caIndex string) (*x509.CertPool, error) {
	info, err := h.getInfoFromLoader()
	if err != nil {
		return nil, err
	}
	caBytes := []byte(info.CAPem)
	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caBytes); !ok {
		return nil, fmt.Errorf("load ca certificate error: no certificate")
	}
	return pool, nil
}

func (h *hooks) ServerHandshakeVerify(cfg *tls.Config) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return nil
}

func (h *hooks) ClientHandshakeVerify(cfg *tls.Config) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return nil
}

func (h *hooks) GenerateHashValue(cfg *tls.Config) *types.HashValue {
	return nil
}

func (h *hooks) GetClientAuth(cfg *v2.TLSConfig) tls.ClientAuthType {
	return tls.NoClientCert
}
