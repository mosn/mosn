package tls

import (
	"crypto/tls"
	"crypto/x509"
	"net"

	"errors"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
	"io/ioutil"
	"strings"
	"sync"
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
	contextMap []map[string]*Context

	isClient  bool
	inspector bool
	context   *Context
	listener  types.Listener

	sync.RWMutex
}

func BuildContextMap(cm *ContextManager, context *Context, index int) {
	if context == nil {
		if index >= len(cm.contextMap) {
			cm.contextMap = append(cm.contextMap, nil)
		} else {
			cm.contextMap[index] = nil
		}
		return
	}

	c := context.tlsConfig

	m := make(map[string]*Context)

	for i := range c.Certificates {
		cert := &c.Certificates[i]
		x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
		if err != nil {
			continue
		}
		if len(x509Cert.Subject.CommonName) > 0 {
			m[x509Cert.Subject.CommonName] = context
		}
		for _, san := range x509Cert.DNSNames {
			m[san] = context
		}
	}

	if context.ALPN != nil {
		for _, protocol := range context.ALPN {
			m[protocol] = context
		}
	}

	m[context.ServerName] = context

	if index >= len(cm.contextMap) {
		cm.contextMap = append(cm.contextMap, m)
	} else {
		cm.contextMap[index] = m
	}
}

func NewTLSServerContextManager(config []v2.FilterChain, l types.Listener) types.TLSContextManager {
	cm := new(ContextManager)

	first := true
	for i, c := range config {
		context, err := NewTLSContext(&c.TLS, cm)
		if err != nil {
			log.StartLogger.Fatalln("New Server TLS Context Manager failed: ", err)
		}

		BuildContextMap(cm, context, i)

		if context != nil {
			context.Listener = l
		}

		if first && context != nil {
			cm.context = context
			first = false
		}
	}

	return cm
}

func AddTLSServerContext(c *v2.TLSConfig, tlsMng types.TLSContextManager, index int) error {
	var cm *ContextManager

	if tlsMng == nil {
		return errors.New("Add Server TLS Context failed: tlsMng is nil")
	}

	if c, ok := tlsMng.(*ContextManager); ok {
		cm = c
	} else {
		return errors.New("Add Server TLS Context failed: tlsMng is not tls.ContextManager")
	}

	if index < 0 || index > len(cm.contextMap) {
		return errors.New("Add Server TLS Context failed: index is out of bounds")
	}

	context, err := NewTLSContext(c, cm)
	if err != nil {
		return err
	}

	context.Listener = cm.listener

	//todo sync.RWMutex, Maps are not safe for concurrent use
	BuildContextMap(cm, context, index)

	if cm.context == nil {
		cm.context = context
	}

	return nil
}

func DelTLSServerContext(tlsMng types.TLSContextManager, index int) error {
	var cm *ContextManager
	if tlsMng == nil {
		return errors.New("Del Server TLS Context failed: tlsMng is nil")
	}

	if c, ok := tlsMng.(*ContextManager); ok {
		cm = c
	} else {
		return errors.New("Del Server TLS Context failed: tlsMng is not tls.ContextManager")
	}

	if index < 0 || index >= len(cm.contextMap) {
		return errors.New("Del Server TLS Context failed: index is out of bounds")
	}

	maps := make([]map[string]*Context, 0)
	for i, m := range cm.contextMap {
		if i != index {
			maps = append(maps, m)
		}
	}

	cm.contextMap = maps

	return nil
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

	for _, maps := range cm.contextMap {
		if maps == nil {
			continue
		}
		// first match ServerName
		// e.g. www.example.com will be first matched against www.example.com, then *.example.com, then *.com
		if info.ServerName != "" {
			name := strings.ToLower(info.ServerName)
			for len(name) > 0 && name[len(name)-1] == '.' {
				name = name[:len(name)-1]
			}

			if context, ok = maps[name]; ok {
				goto find
			}

			labels := strings.Split(name, ".")
			for i := 0; i < len(labels)-1; i++ {
				labels[i] = "*"
				candidate := strings.Join(labels[i:], ".")
				if context, ok = maps[candidate]; ok {
					goto find
				}
			}
		}

		// Sencond match ALPN
		if info.SupportedProtos != nil {
			for _, protocol := range info.SupportedProtos {
				protocol = strings.ToLower(protocol)
				if context, ok = maps[protocol]; ok {
					goto find
				}
			}
		}
	}

	// Last, return the first certificate.
	context = cm.context

find:

	// todo
	// callback select filter config
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
	context := cm.defaultContext()

	if cm.isClient {
		return tls.Client(c, context.tlsConfig)
	}

	if !cm.inspector {
		return tls.Server(c, context.tlsConfig)
	}

	conn := &Conn{
		Conn: c,
	}

	buf := conn.Peek()
	if buf == nil {
		return tls.Server(conn, context.tlsConfig)
	}

	switch buf[0] {
	// TLS handshake
	case 0x16:
		return tls.Server(conn, context.tlsConfig)
	// http plain
	default:
		return conn
	}
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
		var cert tls.Certificate
		var err error

		if strings.Contains(c.CertChain, "-----BEGIN") &&
			strings.Contains(c.PrivateKey, "-----BEGIN") {
			cert, err = tls.X509KeyPair([]byte(c.CertChain), []byte(c.PrivateKey))
			if err != nil {
				log.StartLogger.Fatalln("load [certchain] or [privatekey] error: ", err)
			}
		} else {
			cert, err = tls.LoadX509KeyPair(c.CertChain, c.PrivateKey)
			if err != nil {
				log.StartLogger.Fatalln("load [certchain] or [privatekey] error: ", err)
			}
		}

		context.Certificates = append(context.Certificates, cert)
	}

	if !cm.isClient && len(context.Certificates) == 0 {
		log.StartLogger.Fatalln("[certchain] and [privatekey] are required in TLS config")
	}

	if c.CACert != "" {
		var err error
		var ca []byte

		if strings.Contains(c.CACert, "-----BEGIN") {
			ca = []byte(c.CACert)
		} else {
			ca, err = ioutil.ReadFile(c.CACert)
			if err != nil {
				log.StartLogger.Fatalln("load [cacert] error: ", err)
			}
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

	if c.Inspector {
		cm.inspector = true
	}

	if err := context.NewTLSConfig(cm); err != nil {
		return nil, err
	}

	return context, nil
}

type Conn struct {
	net.Conn
	peek    [1]byte
	haspeek bool
}

func (c *Conn) Peek() []byte {
	b := make([]byte, 1, 1)
	n, err := c.Conn.Read(b)
	if n == 0 {
		log.StartLogger.Infof("TLS Peek() error: ", err)
		return nil
	}

	c.peek[0] = b[0]
	c.haspeek = true
	return b
}

func (c *Conn) Read(b []byte) (int, error) {
	peek := 0
	if c.haspeek {
		c.haspeek = false
		b[0] = c.peek[0]
		if len(b) == 1 {
			return 1, nil
		}

		peek = 1
		b = b[peek:]
	}

	n, err := c.Conn.Read(b)
	return n + peek, err
}
