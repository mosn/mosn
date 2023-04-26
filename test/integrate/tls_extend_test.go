package integrate

import (
	"crypto/x509"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
	mosntls "mosn.io/mosn/pkg/mtls"
	"mosn.io/mosn/pkg/mtls/certtool"
	"mosn.io/mosn/pkg/mtls/crypto/tls"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/protocol/xprotocol/bolt"
	"mosn.io/mosn/pkg/protocol/xprotocol/dubbo"
	"mosn.io/mosn/pkg/types"
	testutil "mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
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

type tlsExtendCase struct {
	*TestCase
}

func (c *tlsExtendCase) Start(conf *testutil.ExtendVerifyConfig) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := testutil.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := testutil.CurrentMeshAddr()
	cfg := testutil.CreateTLSExtensionConfig(clientMeshAddr, serverMeshAddr, c.AppProtocol, c.MeshProtocol, []string{appAddr}, conf)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func TestTLSExtend(t *testing.T) {
	// init extension
	root := certtool.GetRootCA()
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(root.CertPem))
	cert, err := createCert()
	if err != nil {
		t.Error("create certificate failed")
		return
	}
	factory := &tlsConfigHooksFactory{pool, cert}
	extendConfig := &testutil.ExtendVerifyConfig{
		ExtendType: "test",
	}
	if err := mosntls.Register(extendConfig.ExtendType, factory); err != nil {
		t.Errorf("register factory failed %v", err)
		return
	}
	appaddr := "127.0.0.1:8080"
	testCases := []*tlsExtendCase{
		&tlsExtendCase{NewTestCase(t, protocol.HTTP1, protocol.HTTP1, testutil.NewHTTPServer(t, nil))},
		//	&tlsExtendCase{NewTestCase(t, protocol.HTTP1, protocol.HTTP2, testutil.NewHTTPServer(t, nil))},
		//	&tlsExtendCase{NewTestCase(t, protocol.HTTP2, protocol.HTTP1, testutil.NewUpstreamHTTP2(t, appaddr, nil))},
		&tlsExtendCase{NewTestCase(t, protocol.HTTP2, protocol.HTTP2, testutil.NewUpstreamHTTP2(t, appaddr, nil))},
		// protocol auto
		&tlsExtendCase{NewTestCase(t, protocol.HTTP2, protocol.Auto, testutil.NewUpstreamHTTP2(t, appaddr, nil))},
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(extendConfig)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v tls extension test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		tc.FinishCase()
	}
}

type tlsXExtendCase struct {
	*XTestCase
}

func (c *tlsXExtendCase) Start(conf *testutil.ExtendVerifyConfig) {
	c.AppServer.GoServe()
	appAddr := c.AppServer.Addr()
	clientMeshAddr := testutil.CurrentMeshAddr()
	c.ClientMeshAddr = clientMeshAddr
	serverMeshAddr := testutil.CurrentMeshAddr()
	cfg := testutil.CreateXprotocolTLSExtensionConfig(clientMeshAddr, serverMeshAddr, c.SubProtocol, []string{appAddr}, conf)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.Finish
		c.AppServer.Close()
		mesh.Close()
		c.Finish <- true
	}()
	time.Sleep(5 * time.Second) //wait server and mesh start
}

func TestXTLSExtend(t *testing.T) {
	// init extension
	root := certtool.GetRootCA()
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM([]byte(root.CertPem))
	cert, err := createCert()
	if err != nil {
		t.Error("create certificate failed")
		return
	}
	factory := &tlsConfigHooksFactory{pool, cert}
	extendConfig := &testutil.ExtendVerifyConfig{
		ExtendType: "test",
	}
	// ignore already register error
	mosntls.Register(extendConfig.ExtendType, factory)
	appaddr := "127.0.0.1:8080"
	testCases := []*tlsXExtendCase{
		&tlsXExtendCase{NewXTestCase(t, bolt.ProtocolName, testutil.NewRPCServer(t, appaddr, bolt.ProtocolName))},
		&tlsXExtendCase{NewXTestCase(t, dubbo.ProtocolName, testutil.NewRPCServer(t, appaddr, dubbo.ProtocolName))},
		//TODO: boltv2, dubbo, tars
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(extendConfig)
		go tc.RunCase(1, 0)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v tls extension test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		tc.FinishCase()
	}
}
