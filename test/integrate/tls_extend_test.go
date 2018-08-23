package integrate

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	mosntls "github.com/alipay/sofa-mosn/pkg/tls"
	"github.com/alipay/sofa-mosn/pkg/tls/certtool"
	testutil "github.com/alipay/sofa-mosn/test/util"
)

// Test tls config hooks extension
// use tls/util to create certificate
// just verify ca only, ignore the san(dns\ip) verify
type tlsConfigHooks struct {
	root *x509.CertPool
	cert tls.Certificate
}

func (hook *tlsConfigHooks) verifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	var certs []*x509.Certificate
	for _, asn1Data := range rawCerts {
		cert, err := x509.ParseCertificate(asn1Data)
		if err != nil {
			return err
		}
		certs = append(certs, cert)
	}
	intermediates := x509.NewCertPool()
	for _, cert := range certs[1:] {
		intermediates.AddCert(cert)
	}
	opts := x509.VerifyOptions{
		Roots:         hook.root,
		Intermediates: intermediates,
	}
	leaf := certs[0]
	_, err := leaf.Verify(opts)
	return err

}

func (hook *tlsConfigHooks) GetCertificate(certIndex, keyIndex string) (tls.Certificate, error) {
	return hook.cert, nil
}
func (hook *tlsConfigHooks) GetX509Pool(caIndex string) (*x509.CertPool, error) {
	return hook.root, nil
}
func (hook *tlsConfigHooks) VerifyPeerCertificate() func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	return hook.verifyPeerCertificate
}

type tlsConfigHooksFactory struct {
	root *x509.CertPool
	cert tls.Certificate
}

func (f *tlsConfigHooksFactory) CreateConfigHooks(config map[string]interface{}) mosntls.ConfigHooks {
	return &tlsConfigHooks{
		f.root,
		f.cert,
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
	*testCase
}

func (c *tlsExtendCase) Start(conf *testutil.ExtendVerifyConfig) {
	c.appServer.GoServe()
	appAddr := c.appServer.Addr()
	clientMeshAddr := testutil.CurrentMeshAddr()
	c.clientMeshAddr = clientMeshAddr
	serverMeshAddr := testutil.CurrentMeshAddr()
	cfg := testutil.CreateTLSExtensionConfig(clientMeshAddr, serverMeshAddr, c.AppProtocol, c.MeshProtocol, []string{appAddr}, conf)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-c.stop
		c.appServer.Close()
		mesh.Close()
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
		&tlsExtendCase{newTestCase(t, protocol.HTTP1, protocol.HTTP1, testutil.NewHTTPServer(t))},
		&tlsExtendCase{newTestCase(t, protocol.HTTP1, protocol.HTTP2, testutil.NewHTTPServer(t))},
		&tlsExtendCase{newTestCase(t, protocol.HTTP2, protocol.HTTP1, testutil.NewUpstreamHTTP2(t, appaddr))},
		&tlsExtendCase{newTestCase(t, protocol.HTTP2, protocol.HTTP2, testutil.NewUpstreamHTTP2(t, appaddr))},
		&tlsExtendCase{newTestCase(t, protocol.SofaRPC, protocol.HTTP1, testutil.NewRPCServer(t, appaddr, testutil.Bolt1))},
		&tlsExtendCase{newTestCase(t, protocol.SofaRPC, protocol.HTTP2, testutil.NewRPCServer(t, appaddr, testutil.Bolt1))},
		&tlsExtendCase{newTestCase(t, protocol.SofaRPC, protocol.SofaRPC, testutil.NewRPCServer(t, appaddr, testutil.Bolt1))},
	}
	for i, tc := range testCases {
		t.Logf("start case #%d\n", i)
		tc.Start(extendConfig)
		go tc.RunCase(1)
		select {
		case err := <-tc.C:
			if err != nil {
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v tls extension test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
		}
		close(tc.stop)
		time.Sleep(time.Second)
	}
}
