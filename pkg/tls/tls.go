package tls

import (
	"crypto/tls"
	"crypto/x509"
	"net"

	"bufio"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"io/ioutil"
	"strings"
)

var TLSdefaultMinProtocols uint16 = tls.VersionTLS10
var TLSdefaultMaxProtocols uint16 = tls.VersionTLS12

var TLSProtocols = map[string]uint16{
	"tls_auto": 0,
	"tlsv1_0":  tls.VersionTLS10,
	"tlsv1_1":  tls.VersionTLS11,
	"tlsv1_2":  tls.VersionTLS12,
}

var TLSdefaultCurves = []tls.CurveID{
	tls.X25519,
	tls.CurveP256,
}

var TLSALPN = map[string]bool{
	"h2":       true,
	"http/1.1": true,
	"sofa":     true,
}

var TLSCurves = map[string]tls.CurveID{
	"x25519": tls.X25519,
	"p256":   tls.CurveP256,
	"p384":   tls.CurveP384,
	"p521":   tls.CurveP521,
}

var TLSdefaultCiphers = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	tls.TLS_RSA_WITH_AES_128_CBC_SHA,
}

var TLSCiphersMap = map[string]uint16{
	"ECDHE-ECDSA-AES256-GCM-SHA384":      tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	"ECDHE-RSA-AES256-GCM-SHA384":        tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"ECDHE-ECDSA-AES128-GCM-SHA256":      tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"ECDHE-RSA-AES128-GCM-SHA256":        tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"ECDHE-ECDSA-WITH-CHACHA20-POLY1305": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
	"ECDHE-RSA-WITH-CHACHA20-POLY1305":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	"ECDHE-RSA-AES256-CBC-SHA":           tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	"ECDHE-RSA-AES128-CBC-SHA":           tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	"ECDHE-ECDSA-AES256-CBC-SHA":         tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	"ECDHE-ECDSA-AES128-CBC-SHA":         tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	"RSA-AES256-CBC-SHA":                 tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	"RSA-AES128-CBC-SHA":                 tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	"ECDHE-RSA-3DES-EDE-CBC-SHA":         tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
	"RSA-3DES-EDE-CBC-SHA":               tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
}

type Context struct {
	ServerName   string
	MinVersion   uint16
	MaxVersion   uint16
	ALPN         []string
	ClientAuth   tls.ClientAuthType
	CACert       *x509.CertPool
	CipherSuites []uint16
	EcdhCurves   []tls.CurveID
	Certificates []tls.Certificate
	Ticket       string

	VerifyClient bool
	VerifyServer bool

	Listener    types.Listener
	ClusterInfo types.ClusterInfo

	tlsConfig *tls.Config
}

type ContextManager struct {
	contexts   []*Context
	contextMap map[string]*Context

	isClient bool
	context  *Context
	listener types.Listener
}

func BuildContextMap(cm *ContextManager, context *Context) {
	c := context.tlsConfig

	for i := range c.Certificates {
		cert := &c.Certificates[i]
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			continue
		}
		if len(x509Cert.Subject.CommonName) > 0 && cm.contextMap[x509Cert.Subject.CommonName] == nil {
			cm.contextMap[x509Cert.Subject.CommonName] = context
		}
		for _, san := range x509Cert.DNSNames {
			if cm.contextMap[san] == nil {
				cm.contextMap[san] = context
			}
		}
	}

	if context.ALPN != nil {
		for _, protocol := range context.ALPN {
			if cm.contextMap[protocol] == nil {
				cm.contextMap[protocol] = context
			}
		}
	}

	if cm.contextMap[context.ServerName] == nil {
		cm.contextMap[context.ServerName] = context
	}
}

func NewTLSServerContextManager(config []v2.FilterChain, l types.Listener) types.TLSContextManager {
	cm := new(ContextManager)
	cm.contexts = make([]*Context, 1)
	cm.contextMap = make(map[string]*Context)

	first := true
	for _, c := range config {
		context, err := NewTLSContext(&c.TLS, cm)
		if err != nil {
			log.StartLogger.Fatalln("New Server TLS Context Manager failed: ", err)
		}

		cm.contexts = append(cm.contexts, context)

		if context == nil {
			continue
		}

		context.Listener = l
		BuildContextMap(cm, context)

		if first {
			cm.context = context
			first = false
		}
	}

	return cm
}

func NewTLSClientContextManager(config *v2.TLSConfig, info types.ClusterInfo) types.TLSContextManager {
	cm := new(ContextManager)
	cm.isClient = true

	context, err := NewTLSContext(config, cm)
	if err != nil {
		log.StartLogger.Fatalln("New TLS Client Context Manager failed: ", err)
	}

	if context == nil {
		return cm
	}

	context.ClusterInfo = info
	cm.context = context

	return cm
}

func (cm *ContextManager) GetConfigForClient(info *tls.ClientHelloInfo) (*tls.Config, error) {
	var context *Context
	var ok bool

	// first match ServerName
	// e.g. www.example.com will be first matched against www.example.com, then *.example.com, then *.com
	if info.ServerName != "" {
		name := strings.ToLower(info.ServerName)
		for len(name) > 0 && name[len(name)-1] == '.' {
			name = name[:len(name)-1]
		}

		if context, ok = cm.contextMap[name]; ok {
			goto find
		}

		labels := strings.Split(name, ".")
		for i := 0; i < len(labels)-1; i++ {
			labels[i] = "*"
			candidate := strings.Join(labels[i:], ".")
			if context, ok = cm.contextMap[candidate]; ok {
				goto find
			}
		}
	}

	// Sencond match ALPN
	if info.SupportedProtos != nil {
		for _, protocol := range info.SupportedProtos {
			protocol = strings.ToLower(protocol)
			if context, ok = cm.contextMap[protocol]; ok {
				goto find
			}
		}
	}

	// Third, return the first certificate.
	context = cm.context

find:

	// todo
	// callback select filter config
	// var index int
	// for i, c := range cm.contexts {
	//    if context == c {
	//  	index = i
	//    }
	// }
	// callback(cm.listener, index)

	return context.tlsConfig.Clone(), nil

}

func (cm *ContextManager) Enabled() bool {
	if cm.defaultContext() != nil {
		return true
	} else {
		return false
	}
}

func (cm *ContextManager) Conn(c net.Conn) net.Conn {
	var conn *tls.Conn
	context := cm.defaultContext()

	if cm.isClient {
		conn = tls.Client(c, context.tlsConfig)

	} else {
		conn = tls.Server(c, context.tlsConfig)

	}

	return &Conn{Conn: conn, context: context}
}

func (cm *ContextManager) defaultContext() *Context {
	return cm.context
}

func (c *Context) NewTLSConfig(cm *ContextManager) error {
	config := new(tls.Config)
	config.Certificates = c.Certificates
	config.GetConfigForClient = cm.GetConfigForClient
	config.MaxVersion = c.MaxVersion
	config.MinVersion = c.MinVersion
	config.CipherSuites = c.CipherSuites
	config.CurvePreferences = c.EcdhCurves
	config.NextProtos = c.ALPN
	config.InsecureSkipVerify = true

	if c.VerifyClient {
		config.ClientCAs = c.CACert
		config.ClientAuth = tls.RequireAndVerifyClientCert
	}

	if c.VerifyServer {
		config.RootCAs = c.CACert
		config.InsecureSkipVerify = false
	}

	if cm.isClient {
		config.ServerName = c.ServerName
	}

	c.tlsConfig = config

	return nil
}

func NewTLSContext(c *v2.TLSConfig, cm *ContextManager) (*Context, error) {
	if c.Status == false {
		return nil, nil
	}

	context := new(Context)

	if c.CipherSuites != "" {
		ciphers := strings.Split(c.CipherSuites, ":")
		for _, s := range ciphers {
			cipher, ok := TLSCiphersMap[s]
			if !ok {
				log.StartLogger.Fatalln("error cipher name or cipher not supported: ", s)
			}

			context.CipherSuites = append(context.CipherSuites, cipher)
		}

		if len(context.CipherSuites) == 0 {
			context.CipherSuites = TLSdefaultCiphers
		}
	}

	if c.EcdhCurves != "" {
		curves := strings.Split(c.EcdhCurves, ",")
		for _, s := range curves {
			curve, ok := TLSCurves[strings.ToLower(s)]
			if !ok {
				log.StartLogger.Fatalln("error curve name or curve not supported: ", s)
			}

			context.EcdhCurves = append(context.EcdhCurves, curve)
		}

		if len(context.EcdhCurves) == 0 {
			context.EcdhCurves = TLSdefaultCurves
		}
	}

	if c.MaxVersion != "" {
		protocol, ok := TLSProtocols[strings.ToLower(c.MaxVersion)]
		if !ok {
			log.StartLogger.Fatalln("error protocols name or protocol not supported: ", c.MaxVersion)
		}

		if protocol == 0 {
			context.MaxVersion = TLSdefaultMaxProtocols

		} else {
			context.MaxVersion = protocol
		}
	} else {
		context.MaxVersion = TLSdefaultMaxProtocols
	}

	if c.MinVersion != "" {
		protocol, ok := TLSProtocols[strings.ToLower(c.MinVersion)]
		if !ok {
			log.StartLogger.Fatalln("error protocols name or protocol not supported: ", c.MinVersion)
		}

		if protocol == 0 {
			context.MinVersion = TLSdefaultMinProtocols

		} else {
			context.MinVersion = protocol
		}
	} else {
		context.MinVersion = TLSdefaultMinProtocols
	}

	if c.ALPN != "" {
		protocols := strings.Split(c.ALPN, ",")
		for _, p := range protocols {
			_, ok := TLSALPN[strings.ToLower(p)]
			if !ok {
				log.StartLogger.Fatalln("error ALPN or ALPN not supported: ", p)
			}
			context.ALPN = append(context.ALPN, p)
		}
	}

	if c.CertChain != "" && c.PrivateKey != "" {
		cert, err := tls.LoadX509KeyPair(c.CertChain, c.PrivateKey)
		if err != nil {
			log.StartLogger.Fatalln("load [certchain] or [privatekey] error: ", err)
		}
		context.Certificates = append(context.Certificates, cert)
	}

	if !cm.isClient && len(context.Certificates) == 0 {
		log.StartLogger.Fatalln("[certchain] and [privatekey] are required in TLS config")
	}

	if c.CACert != "" {
		ca, err := ioutil.ReadFile(c.CACert)
		if err != nil {
			log.StartLogger.Fatalln("load [cacert] error: ", err)
		}

		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(ca); !ok {
			log.StartLogger.Fatalln("parse [cacert] error")
		}

		context.CACert = pool
	}

	context.VerifyClient = c.VerifyClient
	context.VerifyServer = c.VerifyServer

	context.ServerName = c.ServerName

	if err := context.NewTLSConfig(cm); err != nil {
		return nil, err
	}

	return context, nil
}

type Conn struct {
	net.Conn
	context *Context
	r       *bufio.Reader
}

//func (c *Conn) Peek(n int) ([]byte, error) {
//	return c.r.Peek(n)
//}

//func (c *Conn) Read(b []byte) (int, error) {
//	return c.r.Read(b)
//}

func (c *Conn) SNI() string {
	return ""
}

func (c *Conn) ALPN() string {
	return ""
}
