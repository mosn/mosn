package main

import (
	"crypto/x509"
	"time"

	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	mosntls "mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/mtls/certtool"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
	"mosn.io/mosn/pkg/types"
)

// Test tls config hooks extension
// use tls/util to create certificate
// just verify ca only, ignore the san(dns\ip) verify
type tlsConfigHooks struct {
	root        *x509.CertPool
	cert        tls.Certificate
	defaultHook mosntls.ConfigHooks
}

func (hook *tlsConfigHooks) verifyPeerCertificate(roots *x509.CertPool, certs []*x509.Certificate, t time.Time) error {
	intermediates := x509.NewCertPool()
	for _, cert := range certs[1:] {
		intermediates.AddCert(cert)
	}
	opts := x509.VerifyOptions{
		Roots:         roots,
		CurrentTime:   t,
		Intermediates: intermediates,
	}
	leaf := certs[0]
	_, err := leaf.Verify(opts)
	return err

}

func (hook *tlsConfigHooks) GetClientAuth(cfg *v2.TLSConfig) tls.ClientAuthType {
	return hook.defaultHook.GetClientAuth(cfg)
}

func (hook *tlsConfigHooks) GenerateHashValue(cfg *tls.Config) *types.HashValue {
	return hook.defaultHook.GenerateHashValue(cfg)
}

func (hook *tlsConfigHooks) GetCertificate(certIndex, keyIndex string) (tls.Certificate, error) {
	return hook.cert, nil
}

func (hook *tlsConfigHooks) GetX509Pool(caIndex string) (*x509.CertPool, error) {
	return hook.root, nil
}

func (hook *tlsConfigHooks) ServerHandshakeVerify(cfg *tls.Config) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		certs := make([]*x509.Certificate, 0, len(rawCerts))
		for _, asn1Data := range rawCerts {
			cert, err := x509.ParseCertificate(asn1Data)
			if err != nil {
				return err
			}
			certs = append(certs, cert)
		}
		if cfg.ClientAuth >= tls.VerifyClientCertIfGiven && len(certs) > 0 {
			var t time.Time
			if cfg.Time != nil {
				t = cfg.Time()
			} else {
				t = time.Now()
			}
			return hook.verifyPeerCertificate(cfg.ClientCAs, certs, t)
		}
		return nil
	}
}

func (hook *tlsConfigHooks) ClientHandshakeVerify(cfg *tls.Config) func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		certs := make([]*x509.Certificate, 0, len(rawCerts))
		for _, asn1Data := range rawCerts {
			cert, err := x509.ParseCertificate(asn1Data)
			if err != nil {
				return err
			}
			certs = append(certs, cert)
		}
		var t time.Time
		if cfg.Time != nil {
			t = cfg.Time()
		} else {
			t = time.Now()
		}
		return hook.verifyPeerCertificate(cfg.RootCAs, certs, t)
	}
}

type tlsConfigHooksFactory struct {
	root *x509.CertPool
	cert tls.Certificate
}

func (f *tlsConfigHooksFactory) CreateConfigHooks(config map[string]interface{}) mosntls.ConfigHooks {
	return &tlsConfigHooks{
		f.root,
		f.cert,
		mosntls.DefaultConfigHooks(),
	}
}

func createCert() (tls.Certificate, error) {
	var cert tls.Certificate
	priv, err := certtool.GeneratePrivateKey("P256")
	if err != nil {
		return cert, err
	}
	tmpl, err := certtool.CreateTemplate("test", false, nil)
	if err != nil {
		return cert, err
	}
	// No SAN
	tmpl.IPAddresses = nil
	c, err := certtool.SignCertificate(tmpl, priv)
	if err != nil {
		return cert, err
	}
	return tls.X509KeyPair([]byte(c.CertPem), []byte(c.KeyPem))
}

func init() {
	// init extension
	root := certtool.GetRootCA()
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(root.CertPem))
	cert, err := createCert()
	if err != nil {
		log.DefaultLogger.Fatalf("init tls extension failed: %v", err)
	}
	factory := &tlsConfigHooksFactory{pool, cert}
	if err := mosntls.Register("mosn-integrate-test-tls", factory); err != nil {
		log.DefaultLogger.Fatalf("register tls extension failed: %v", err)
	}
}
