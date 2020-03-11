package tls

// #include "shim.h"
import "C"
import (
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"runtime"
	"sync"
	"unsafe"

	"mosn.io/mosn/pkg/mtls/crypto/x509"
)

// SslCertificate record cert information of ssl, actually it would be set into
// tls.Certificate when read x509keypair
type SslCertificate struct {
	Cert      *C.X509
	CertChain *C.struct_stack_st_X509
	Pkey      *C.EVP_PKEY
}

// SslCtx record *SSL_CTX and some neccessary info about handshake, it would be init
// when call Client() or Server()
type SslCtx struct {
	sslCtx     *C.SSL_CTX
	ServerName string
	Conn       net.Conn
}

// SslConnectionState is use to get ssl handshake state, the most difference from
// tls.ConnectionState is Certificate, SslConnectionState would record openssl X509*
// cert in struct
type SslConnectionState struct {
	Version                     uint16              // TLS version used by the connection (e.g. VersionTLS12)
	HandshakeComplete           bool                // TLS handshake is complete
	DidResume                   bool                // connection resumes a previous TLS connection
	CipherSuite                 uint16              // cipher suite in use (TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, ...)
	NegotiatedProtocol          string              // negotiated next protocol (not guaranteed to be from Config.NextProtos)
	NegotiatedProtocolIsMutual  bool                // negotiated protocol was advertised by server (client side only)
	ServerName                  string              // server name requested by client, if any (server side only)
	PeerCertificates            []*x509.Certificate // certificate chain presented by remote peer
	VerifiedChains              **C.X509            // verified chains built from PeerCertificates
	SignedCertificateTimestamps [][]byte            // SCTs from the peer, if any
	OCSPResponse                []byte              // stapled OCSP response from peer, if any

	// ekm is a closure exposed via ExportKeyingMaterial.
	ekm func(label string, context []byte, length int) ([]byte, error)

	// TLSUnique contains the "tls-unique" channel binding value (see RFC
	// 5929, section 3). For resumed sessions this value will be nil
	// because resumption does not include enough context (see
	// https://mitls.org/pages/attacks/3SHAKE#channelbindings). This will
	// change in future versions of Go once the TLS master-secret fix has
	// been standardized and implemented. It is not defined in TLS 1.3.
	TLSUnique []byte
}

// Ssl record *SSL and if has finish handshake, it would be init when calling Handshake()
type Ssl struct {
	mtx               sync.Mutex
	handshakeComplete uint32
	ssl               *C.SSL
	readBio           ReadBio
	writeBio          WriteBio
	conn              net.Conn
	connState         SslConnectionState
	isClosed          bool
}

// GetSslCtx return openssl ssl_ctx* of SslCtx struct
func (ctx *SslCtx) GetSslCtx() *C.SSL_CTX {
	if ctx != nil {
		return ctx.sslCtx
	}
	return nil
}

// GetSsl return openssl ssl* of Ssl struct
func (ssl *Ssl) GetSsl() *C.SSL {
	return ssl.ssl
}

func (ctx *SslCtx) serverSslCtxInit(config *Config) error {
	var ret int

	sslCtx := C.SSL_CTX_new(C.TLS_server_method())
	if sslCtx == nil {
		return errors.New("SSL_CTX_new error")
	}

	if len(config.Certificates) > 0 {
		sslCert := config.Certificates[0].BabasslCert
		err := setSslCertAndPkeyToSslCtx(sslCtx, sslCert)
		if err != nil {
			return err
		}
	}

	if config.NextProtos != nil {
		err := serverSslCtxSetAlpnProtos(sslCtx, config.NextProtos)
		if err != nil {
			return err
		}
	}

	err := SslCtxSetDefaultCipher(sslCtx)
	if err != nil {
		return err
	}

	if config.CipherSuites != nil {
		err = SslCtxSetCiphersuites(sslCtx, config.CipherSuites)
		if err != nil {
			return err
		}
	}

	//SSL_CTX_set_min_proto_version(ctx, min_version)
	ret = int(C.SSL_CTX_ctrl(sslCtx, C.SSL_CTRL_SET_MIN_PROTO_VERSION, C.long(config.MinVersion), C.NULL))
	if ret <= 0 {
		return errors.New("SSL_CTX_set_min_proto_version error")
	}
	//SSL_CTX_set_max_proto_version(ctx, max_version)
	ret = int(C.SSL_CTX_ctrl(sslCtx, C.SSL_CTRL_SET_MAX_PROTO_VERSION, C.long(config.MaxVersion), C.NULL))
	if ret <= 0 {
		return errors.New("SSL_CTX_set_max_proto_version error")
	}

	if config.GetConfigForClient != nil {
		ConfigForClientPtr := cgoPointerSave(config)
		C.ssl_ctx_set_client_hello_cb_GetConfigForClient(sslCtx, ConfigForClientPtr)
	}

	if config.ClientAuth == RequestClientCert || config.ClientAuth == RequireAnyClientCert {
		C.ssl_ctx_set_cert_verify_require_peer_cert(sslCtx)
	} else if config.ClientAuth >= VerifyClientCertIfGiven {
		C.SSL_CTX_set_verify(sslCtx, C.SSL_VERIFY_PEER, nil)
	} else if config.ClientAuth == RequireAndVerifyClientCert {
		C.SSL_CTX_set_verify(sslCtx, C.SSL_VERIFY_PEER|C.SSL_VERIFY_FAIL_IF_NO_PEER_CERT, nil)
	}

	if config.ClientCAs != nil {
		clientCAS, err := TranslateGoCertsToSslX509s(config.ClientCAs.GetCerts())
		if err != nil {
			return err
		}
		err = SetVerifyCertsIntoCtx(sslCtx, clientCAS)
		if err != nil {
			return err
		}
	}

	if config.VerifyPeerCertificate != nil {
		configPtr := cgoPointerSave(config)
		C.ssl_ctx_set_cert_verify_callback_serverVerifyBackForVerifyPeerCertificate(sslCtx, configPtr)
	}

	ctx.sslCtx = sslCtx
	return nil
}

func (ctx *SslCtx) clientSslCtxInit(config *Config) error {
	var err error
	var verifyMode C.int
	sslCtx := C.SSL_CTX_new(C.TLS_client_method())
	verifyMode = C.SSL_VERIFY_PEER
	if !config.InsecureSkipVerify {
		verifyMode = C.SSL_VERIFY_PEER
	} else {
		verifyMode = C.SSL_VERIFY_NONE
	}

	if config.ServerName != "" {
		servername := hostnameInSNI(config.ServerName)
		clientSetSslCtxServerNameCallBack(sslCtx, servername)
		ctx.ServerName = servername
	}

	C.SSL_CTX_set_verify(sslCtx, verifyMode, nil)

	if config.RootCAs != nil {
		rootCAS, tranErr := TranslateGoCertsToSslX509s(config.RootCAs.GetCerts())
		if tranErr != nil {
			return tranErr
		}
		err = SetVerifyCertsIntoCtx(sslCtx, rootCAS)
		if err != nil {
			return err
		}
	}

	//C.SSL_CTX_set_mode(sslCtx, C.SSL_MODE_AUTO_RETRY)
	C.SSL_CTX_ctrl(sslCtx, C.SSL_CTRL_MODE, C.SSL_MODE_AUTO_RETRY, C.NULL)
	if config.NextProtos != nil {
		err = clientSslCtxSetAlpnProtos(sslCtx, config.NextProtos)
		if err != nil {
			return err
		}
	}

	if config.CipherSuites != nil {
		err := SslCtxSetCiphersuites(sslCtx, config.CipherSuites)
		if err != nil {
			return err
		}
	}

	//SSL_CTX_set_min_proto_version(ctx, min_version)
	ret := int(C.SSL_CTX_ctrl(sslCtx, C.SSL_CTRL_SET_MIN_PROTO_VERSION, C.long(config.MinVersion), C.NULL))
	if ret <= 0 {
		return errors.New("SSL_CTX_set_min_proto_version error")
	}
	//SSL_CTX_set_max_proto_version(ctx, max_version)
	ret = int(C.SSL_CTX_ctrl(sslCtx, C.SSL_CTRL_SET_MAX_PROTO_VERSION, C.long(config.MaxVersion), C.NULL))
	if ret <= 0 {
		return errors.New("SSL_CTX_set_max_proto_version error")
	}

	if len(config.Certificates) > 0 {
		sslCert := config.Certificates[0].BabasslCert
		err := setSslCertAndPkeyToSslCtx(sslCtx, sslCert)
		if err != nil {
			return err
		}
	}

	ctx.sslCtx = sslCtx
	return nil
}

// Init is used to init a SslCtx, the core work is to use go native Config
// to initialize openssl ssl_ctx*, like set cert, privatekey, call back and so on
func (ctx *SslCtx) Init(config *Config, isClient bool) error {
	runtime.SetFinalizer(ctx, func(ctx *SslCtx) {
		C.SSL_CTX_free(ctx.sslCtx)
	})

	if isClient {
		return ctx.clientSslCtxInit(config)
	}

	return ctx.serverSslCtxInit(config)
}

// func (ctx *SslCtx) Clone() *SslCtx {
// 	if ctx == nil {
// 		return nil
// 	}
// 	sslCtxNew := C.SSL_CTX_dup(ctx.sslCtx)
// 	if unsafe.Pointer(sslCtxNew) == C.NULL {
// 		panic("SSL_CTX_dup error")
// 	}
// 	ctxNew := &SslCtx{
// 		sslCtx: sslCtxNew,
// 	}
// 	runtime.SetFinalizer(ctxNew, func(ctxNew *SslCtx) {
// 		C.SSL_CTX_free(ctxNew.sslCtx)
// 	})
// 	return ctxNew
// }

func defaultClientSslCtx() *SslCtx {
	sslCtx := C.SSL_CTX_new(C.TLS_client_method())
	if sslCtx == nil {
		return nil
	}
	C.SSL_CTX_set_verify(sslCtx, C.SSL_VERIFY_NONE, nil)
	//C.SSL_CTX_set_mode(sslCtx, C.SSL_MODE_AUTO_RETRY)
	C.SSL_CTX_ctrl(sslCtx, C.SSL_CTRL_MODE, C.SSL_MODE_AUTO_RETRY, C.NULL)
	ctx := &SslCtx{
		sslCtx: sslCtx,
	}
	return ctx
}

func (ssl *Ssl) clientSslInit(conn net.Conn, sslCtx *SslCtx) error {
	if sslCtx == nil || sslCtx.GetSslCtx() == nil {
		sslCtx = defaultClientSslCtx()
		if sslCtx == nil {
			return errors.New("make defaultClientSslCtx error")
		}
	}
	s := C.SSL_new(sslCtx.GetSslCtx())
	if s == nil {
		return errors.New("ssl new error")
	}
	if rc := C.X_shim_init(); rc != 0 {
		panic("X_shim_init failed with %d")
	}

	ssl.readBio = ReadBio{}
	ssl.writeBio = WriteBio{}

	intoSslCbio := ssl.readBio.MakeCBIO()
	fromSslCbio := ssl.writeBio.MakeCBIO()

	ssl.readBio.conn = conn
	ssl.writeBio.conn = conn
	ssl.readBio.mtx = &ssl.mtx
	ssl.writeBio.mtx = &ssl.mtx
	if intoSslCbio == nil || fromSslCbio == nil {
		// these frees are null safe
		C.BIO_free(intoSslCbio)
		C.BIO_free(fromSslCbio)
		C.SSL_free(s)
		return errors.New("failed to allocate memory BIO")
	}

	C.SSL_set_bio(s, intoSslCbio, fromSslCbio)
	C.SSL_set_connect_state(s)

	if sslCtx.ServerName != "" {
		clientSslSetServerName(s, sslCtx.ServerName)
	}
	ssl.ssl = s
	ssl.conn = conn
	return nil
}

func (ssl *Ssl) serverSslInit(conn net.Conn, sslCtx *SslCtx) error {
	s := C.SSL_new(sslCtx.GetSslCtx())
	if s == nil {
		return errors.New("ssl new error")
	}
	if rc := C.X_shim_init(); rc != 0 {
		panic("X_shim_init failed with %d")
	}

	ssl.readBio = ReadBio{}
	ssl.writeBio = WriteBio{}

	intoSslCbio := ssl.readBio.MakeCBIO()
	fromSslCbio := ssl.writeBio.MakeCBIO()
	if intoSslCbio == nil || fromSslCbio == nil {
		// these frees are null safe
		C.BIO_free(intoSslCbio)
		C.BIO_free(fromSslCbio)
		C.SSL_free(s)
		return errors.New("failed to allocate memory BIO")
	}
	ssl.readBio.conn = conn
	ssl.writeBio.conn = conn
	ssl.readBio.mtx = &ssl.mtx
	ssl.writeBio.mtx = &ssl.mtx

	C.SSL_set_bio(s, intoSslCbio, fromSslCbio)
	C.SSL_set_accept_state(s)

	ssl.ssl = s
	ssl.conn = conn
	return nil
}

// Init of Ssl is to Init openssl ssl*, the core work is to use a initialized
// ssl_ctx* to generate a format ssl*, which is used for later handshake
func (ssl *Ssl) Init(conn net.Conn, sslCtx *SslCtx, isClient bool) error {
	runtime.SetFinalizer(ssl, func(ssl *Ssl) {
		C.SSL_clear(ssl.ssl)
		C.SSL_free(ssl.ssl)
	})
	if isClient {
		return ssl.clientSslInit(conn, sslCtx)
	}

	return ssl.serverSslInit(conn, sslCtx)
}

// Close is used to close ssl connection in openssl
func (ssl *Ssl) Close() {
	ssl.mtx.Lock()
	defer ssl.mtx.Unlock()
	if ssl != nil {
		C.SSL_shutdown(ssl.ssl)
		ssl.isClosed = true
	}
}

// getFdFromConn is used to get accept fd in current conn
func getFdFromConn(conn net.Conn) int {
	val := reflect.Indirect(reflect.ValueOf(conn))
	fdmember := reflect.Indirect(val.FieldByName("fd"))
	pfd := reflect.Indirect(fdmember.FieldByName("pfd"))
	fd := int(pfd.FieldByName("Sysfd").Int())
	return fd
}

// GetRawInput is used to get raw information from peer
func (ssl *Ssl) GetRawInput() []byte {
	return ssl.readBio.getRawInput()
}

// DoHandshake is used to finish whold handshake process in openssl
func (ssl *Ssl) DoHandshake() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ssl.mtx.Lock()
	defer ssl.mtx.Unlock()
	if ssl.isClosed {
		return io.EOF
	}
	for {
		sslErr := C.SSL_do_handshake(ssl.ssl)
		if int(sslErr) <= 0 {
			sslErr = C.SSL_get_error(ssl.ssl, sslErr)
			if ssl.readBio.err != nil {
				return ssl.readBio.err
			}
			if ssl.writeBio.err != nil {
				return ssl.writeBio.err
			}
			switch sslErr {
			case C.SSL_ERROR_WANT_READ:
			case C.SSL_ERROR_WANT_WRITE:
				continue
			case C.SSL_ERROR_ZERO_RETURN:
				return io.EOF
			case C.SSL_ERROR_SSL:
				errReason := getSslError()
				if BabasslPrintTraceTag.IsOpen() {
					print(errReason)
				}
				return sslGenStandardError(sslErrSsl, errReason)
			default:
				if BabasslPrintTraceTag.IsOpen() {
					C.ERR_print_errors_fp(C.stderr)
				}
				return errors.New("BabaSSL handshake error")
			}
		} else {
			break
		}
	}
	ssl.handshakeComplete = 1
	return nil
}

// Read is used to read decoded information from openssl connection to b
func (ssl *Ssl) Read(b []byte) (int, error) {
	if ssl.isClosed {
		return 0, io.EOF
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var err error
	ssl.mtx.Lock()
	defer ssl.mtx.Unlock()
	for {
		if ssl == nil || ssl.ssl == nil {
			panic("ssl is nil")
		}
		sslErr := C.SSL_read(ssl.ssl, unsafe.Pointer(&b[0]), C.int(len(b)))

		if int(sslErr) <= 0 {
			sslErr = C.SSL_get_error(ssl.ssl, sslErr)
			if ssl.readBio.err != nil {
				return 0, ssl.readBio.err
			}
			if ssl.writeBio.err != nil {
				return 0, ssl.writeBio.err
			}
			switch sslErr {
			case C.SSL_ERROR_WANT_READ:
			case C.SSL_ERROR_ZERO_RETURN:
				return 0, io.EOF
			case C.SSL_ERROR_SSL:
				errReason := getSslError()
				if BabasslPrintTraceTag.IsOpen() {
					print(errReason)
				}
				return 0, sslGenStandardError(sslErrSsl, errReason)
			default:
				if BabasslPrintTraceTag.IsOpen() {
					C.ERR_print_errors_fp(C.stderr)
				}
				return 0, errors.New("BabaSSL SSL_read error")
			}
		} else {
			return int(sslErr), err
		}
	}
}

// Write is used to write information for b to peer
func (ssl *Ssl) Write(b []byte) (int, error) {
	if ssl.isClosed {
		return 0, io.EOF
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	totalLen := len(b)
	writeLen := 0
	ssl.mtx.Lock()
	defer ssl.mtx.Unlock()
	for writeLen < totalLen {
		n := C.SSL_write(ssl.ssl, unsafe.Pointer(&b[0]), C.int(len(b)))

		if n > 0 {
			writeLen += int(n)
		} else if n <= 0 {
			sslErr := C.SSL_get_error(ssl.ssl, n)
			if ssl.writeBio.err != nil {
				return 0, ssl.writeBio.err
			}
			if ssl.readBio.err != nil {
				return 0, ssl.readBio.err
			}
			switch sslErr {
			case C.SSL_ERROR_WANT_WRITE:
				continue
			case C.SSL_ERROR_ZERO_RETURN:
				return 0, io.EOF
			case C.SSL_ERROR_SSL:
				errReason := getSslError()
				if BabasslPrintTraceTag.IsOpen() {
					print(errReason)
				}
				return 0, sslGenStandardError(sslErrSsl, errReason)
			default:
				if BabasslPrintTraceTag.IsOpen() {
					C.ERR_print_errors_fp(C.stderr)
				}
				return 0, errors.New("BabaSSL SSL_write error")
			}
		}
	}
	return writeLen, nil
}

// GetConnectionState is used to get current connection state of ssl
func (ssl *Ssl) GetConnectionState() *SslConnectionState {
	if ssl.ssl == nil {
		return nil
	}
	var err error
	connState := SslConnectionState{}
	connState.Version = uint16(C.SSL_version(ssl.ssl))
	session := C.SSL_get_session(ssl.ssl)
	if int(C.SSL_SESSION_is_resumable(session)) == 1 {
		connState.DidResume = true
	} else {
		connState.DidResume = false
	}
	connState.CipherSuite = uint16(C.SSL_CIPHER_get_id(C.SSL_get_current_cipher(ssl.ssl)))
	connState.NegotiatedProtocol, connState.NegotiatedProtocolIsMutual = getSslAlpnNegotiated(ssl.ssl)
	connState.PeerCertificates, err = getPeerCertsAsX509Arr(ssl.ssl)
	if err != nil {
		if BabasslPrintTraceTag.IsOpen() {
			fmt.Println(err)
		}
	}

	ssl.connState = connState
	return &connState

}
