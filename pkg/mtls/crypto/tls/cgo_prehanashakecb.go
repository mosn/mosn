package tls

/*
#include "shim.h"

static int SSL_client_hello_servername_ext_to_gostring(SSL *s, void *gostring)
{
	const unsigned char *p = NULL;
	const char *servername;
	int ret = 0;
	size_t len, remaining;

	ret = SSL_client_hello_get0_ext(s, TLSEXT_TYPE_server_name, &p, &remaining);
	if (ret <= 0 || remaining <= 2) {
		return 0;
	}

	// Extract the length of the supplied list of names.
    len = (*(p++) << 8);
    len += *(p++);
    if (len + 2 != remaining)
        return 0;
    remaining = len;

	// The list in practice only has a single element, so we only consider
    // the first one.
    if (remaining == 0 || *p++ != TLSEXT_NAMETYPE_host_name)
        return 0;
	remaining--;

    //Now we can finally pull out the byte array with the actual hostname.
    if (remaining <= 2)
        return 0;
    len = (*(p++) << 8);
    len += *(p++);
    if (len + 2 > remaining)
        return 0;
    remaining = len;
    servername = (const char *)p;

	memcpy(gostring, servername, remaining);
	return remaining;
}

static int SSL_client_hello_get0_extension_to_gobuf(SSL *s, int type, void *buf)
{
	const unsigned char *p = NULL;
	size_t len = 0;
	int ret = 0;

	ret = SSL_client_hello_get0_ext(s, type, &p, &len);
	if (ret == 0 || len <= 0) {
		return 0;
	}

	memcpy(buf, p, len);
	return len;
}

static int SSL_client_hello_get0_ciphers_to_gobuf(SSL *s, void *buf)
{
	const unsigned char *ciphers;
	size_t len;

	len = SSL_client_hello_get0_ciphers(s, &ciphers);
	memcpy(buf, ciphers, len);

	return len;
}
*/
import "C"
import (
	"crypto/x509"
	"errors"
	"unsafe"
)

type clientHelloCbParam struct {
	getConfigForClient func(*ClientHelloInfo) (*Config, error)
	getCertificate     func(*ClientHelloInfo) (*Certificate, error)
	certs              []x509.Certificate
}

// serverClientHelloCallBackForGetConfigForClient will be called in openssl after server
// read client hello infomation, this call back would call go native call back GetConfigForClient
// and GetCertificate in conn.Config if it is not nil
//export serverClientHelloCallBackForGetConfigForClient
func serverClientHelloCallBackForGetConfigForClient(ssl *C.SSL, al *C.int, arg unsafe.Pointer) C.int {
	conf, ok := cgoPointerRestore(arg).(*Config)
	cgoPointerUnref(arg)
	if !ok {
		return -1
	}

	//transfer babassl info to go-native tls.ClientHelloInfo
	ch := transferBabasslInfoToTLSClientHelloInfo(ssl)
	ch.Conn = conf.CgoBabasslCtx.Conn
	//call go-native GetConfigForClient
	tlsConfig := &Config{}
	var err error
	if conf.GetConfigForClient != nil {
		tlsConfig, err = conf.GetConfigForClient(ch)
		if err != nil {
			if BabasslPrintTraceTag.IsOpen() {
				print(err)
			}
			return -1
		}
	}

	//call go-native GetCertificate
	if conf.GetCertificate != nil {
		cert, err := conf.GetCertificate(ch)
		if err != nil {
			if BabasslPrintTraceTag.IsOpen() {
				print(err)
			}
			return -1
		}
		tlsConfig.Certificates = append(tlsConfig.Certificates, Certificate{})
		copy(tlsConfig.Certificates[1:], tlsConfig.Certificates[:])
		tlsConfig.Certificates[0] = *cert
	}

	//transfer tls.ClientHelloInfo to babassl info and set
	err = setTLSConfigInfoToSsl(ssl, tlsConfig)
	if err != nil {
		return -1
	}
	return 1
}

func setTLSConfigInfoToSsl(ssl *C.SSL, conf *Config) error {
	ctx := C.SSL_get_SSL_CTX(ssl)
	ctxErr := setTLSConfigInfoToSslCtx(ctx, conf)
	if ctxErr != nil {
		return ctxErr
	}

	if len(conf.Certificates) != 0 {
		sslCert := conf.Certificates[0].BabasslCert
		err := setSslCertAndPkeyToSsl(ssl, sslCert)
		if err != nil {
			return err
		}
	}

	if conf.ClientAuth == RequestClientCert || conf.ClientAuth == RequireAnyClientCert {
		C.ssl_set_cert_verify_require_peer_cert(ssl)
	} else if conf.ClientAuth == VerifyClientCertIfGiven {
		C.SSL_set_verify(ssl, C.SSL_VERIFY_PEER, nil)
	} else if conf.ClientAuth == RequireAndVerifyClientCert {
		C.SSL_set_verify(ssl, C.SSL_VERIFY_PEER|C.SSL_VERIFY_FAIL_IF_NO_PEER_CERT, nil)
	}

	return nil
}

func setTLSConfigInfoToSslCtx(ctx *C.SSL_CTX, conf *Config) error {
	configPtr := cgoPointerSave(conf)
	C.ssl_ctx_set_cert_verify_callback_serverCertVerifyCallBack(ctx, configPtr)

	if conf.NextProtos != nil {
		err := serverSslCtxSetAlpnProtos(ctx, conf.NextProtos)
		if err != nil {
			return err
		}
	}
	return nil
}

func transferBabasslInfoToTLSClientHelloInfo(ssl *C.SSL) *ClientHelloInfo {
	ch := &ClientHelloInfo{}
	servernameBuf := make([]byte, 4096)
	servernameLen := int(C.SSL_client_hello_servername_ext_to_gostring(ssl,
		unsafe.Pointer(&servernameBuf[0])))
	if servernameLen > 0 {
		servername := BytesToString(servernameBuf[:servernameLen])
		ch.ServerName = servername
	}

	alpn, err := getClientAlpn(ssl)
	if err == nil {
		if BabasslPrintTraceTag.IsOpen() {
			print(err)
		}
	}
	ch.SupportedProtos = alpn

	cipher, err := getClientCiphersuites(ssl)
	if err == nil {
		if BabasslPrintTraceTag.IsOpen() {
			print(err)
		}
	}
	ch.CipherSuites = cipher
	curves, err := getClientSupportGroup(ssl)
	if err != nil {
		if BabasslPrintTraceTag.IsOpen() {
			print(err)
		}
	}
	ch.SupportedCurves = curves
	ecPointFormat, err := getClientEcPointFormat(ssl)
	if err != nil {
		if BabasslPrintTraceTag.IsOpen() {
			print(err)
		}
	}
	ch.SupportedPoints = ecPointFormat
	sigalg, err := getClientSignatureAlgorithm(ssl)
	if err != nil {
		if BabasslPrintTraceTag.IsOpen() {
			print(err)
		}
	}
	ch.SignatureSchemes = sigalg

	if err != nil {
		if BabasslPrintTraceTag.IsOpen() {
			print(err)
		}
	}
	ch.SupportedProtos = alpn

	version, err := getClientVersion(ssl)
	if err != nil {
		if BabasslPrintTraceTag.IsOpen() {
			print(err)
		}
	}
	ch.SupportedVersions = version
	return ch
}

// serverVerifyBackForVerifyPeerCertificate will be called in openssl after get peer certs,
// this call back would call go native call back VerifyPeerCertificate in conn.Config
// if it is not nil
//export serverVerifyBackForVerifyPeerCertificate
func serverVerifyBackForVerifyPeerCertificate(xs *C.X509_STORE_CTX, arg unsafe.Pointer) C.int {
	conf, ok := cgoPointerRestore(arg).(*Config)
	cgoPointerUnref(arg)
	if !ok {
		return -1
	}

	if conf.VerifyPeerCertificate != nil {
		//get peer cert chain
		untrustChain := C.X509_STORE_CTX_get0_untrusted(xs)
		peerCerts, err := TranslateSslX509StackToGoCerts(untrustChain)
		if err != nil {
			return -1
		}
		peerCertsRaw, rawErr := TranslateSslX509StackToRawBytes(untrustChain)
		if rawErr != nil {
			return -1
		}

		if conf.ClientAuth >= VerifyClientCertIfGiven && len(peerCerts) > 0 {
			opts := x509.VerifyOptions{
				Roots:         conf.ClientCAs,
				CurrentTime:   conf.time(),
				Intermediates: x509.NewCertPool(),
				KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			}

			for _, cert := range peerCerts[1:] {
				opts.Intermediates.AddCert(cert)
			}

			chains, err := peerCerts[0].Verify(opts)
			if err != nil {
				return -1
			}

			if err := conf.VerifyPeerCertificate(peerCertsRaw, chains); err != nil {
				return -1
			}
		}
	}
	return 1
}

func getClientAlpn(ssl *C.SSL) ([]string, error) {
	buf := make([]byte, 1024, 1024)
	len := int(C.SSL_client_hello_get0_extension_to_gobuf(ssl,
		C.TLSEXT_TYPE_application_layer_protocol_negotiation, unsafe.Pointer(&buf[0])))
	if len == 0 {
		return nil, nil
	}
	return parseAlpnFromExtension(buf[:len])
}

func getClientCiphersuites(ssl *C.SSL) ([]uint16, error) {
	buf := make([]byte, 2048, 2048)
	len := int(C.SSL_client_hello_get0_ciphers_to_gobuf(ssl, unsafe.Pointer(&buf[0])))
	//a cipher is consist of 2byte
	if len%2 != 0 {
		return nil, errors.New("getClientCiphersuites error, wrong packet format")
	}
	buf = buf[:len]
	index := 0
	var res []uint16
	var ciph uint16
	for index < len {
		ciph = uint16(buf[index])
		ciph = ciph << 8
		ciph |= uint16(buf[index+1])
		res = append(res, ciph)
		index += 2
	}

	return res, nil
}

func getClientSupportGroup(ssl *C.SSL) ([]CurveID, error) {
	buf := make([]byte, 512, 512)
	len := int(C.SSL_client_hello_get0_extension_to_gobuf(ssl,
		C.TLSEXT_TYPE_supported_groups, unsafe.Pointer(&buf[0])))
	if len == 0 {
		return nil, nil
	}
	if len%2 != 0 {
		return nil, errors.New("getClientSupportGroup error, wrong packet format")
	}
	buf = buf[:len]
	index := 0
	var res []CurveID
	for index < len {
		curve := uint16(buf[index])
		curve = curve << 8
		curve |= uint16(buf[index+1])
		// Golang only support 4 curves(see type CurveID struct), now we  support
		// all in cgo_BabaSSL
		res = append(res, CurveID(curve))
		index += 2
	}

	return res, nil
}

func getClientEcPointFormat(ssl *C.SSL) ([]uint8, error) {
	buf := make([]byte, 16, 16)
	len := int(C.SSL_client_hello_get0_extension_to_gobuf(ssl,
		C.TLSEXT_TYPE_ec_point_formats, unsafe.Pointer(&buf[0])))
	if len == 0 {
		return nil, nil
	}
	buf = buf[:len]
	ecPointFormatLen := int(buf[0])
	if ecPointFormatLen != len-1 {
		return nil, errors.New("getClientEcPointFormat error, wrong packet format")
	}
	var res []uint8
	for i := 0; i < ecPointFormatLen; i++ {
		format := uint8(buf[i+1])
		res = append(res, format)
	}
	return res, nil
}

func getClientSignatureAlgorithm(ssl *C.SSL) ([]SignatureScheme, error) {
	buf := make([]byte, 1024, 1024)
	len := int(C.SSL_client_hello_get0_extension_to_gobuf(ssl,
		C.TLSEXT_TYPE_signature_algorithms, unsafe.Pointer(&buf[0])))
	if len == 0 {
		return nil, nil
	}
	if len < 2 {
		return nil, errors.New("getClientSignatureAlgorithm error, wrong packet format")
	}
	buf = buf[:len]

	totalLen := uint16(buf[0])
	totalLen = totalLen << 8
	totalLen |= uint16(buf[1])
	if totalLen == 0 {
		return nil, nil
	}
	if totalLen%2 != 0 {
		return nil, errors.New("getClientSignatureAlgorithm error, wrong packet format")
	}
	var res []SignatureScheme
	index := uint16(0)
	buf = buf[2:]
	for index < totalLen {
		sig := uint16(buf[index])
		sig = sig << 8
		sig |= uint16(buf[index+1])
		index += 2
		res = append(res, SignatureScheme(sig))
	}
	return res, nil
}

func getClientVersion(ssl *C.SSL) ([]uint16, error) {
	buf := make([]byte, 1024, 1024)
	len := int(C.SSL_client_hello_get0_extension_to_gobuf(ssl,
		C.TLSEXT_TYPE_supported_versions, unsafe.Pointer(&buf[0])))
	if len == 0 {
		return nil, nil
	}
	buf = buf[:len]
	totalLen := uint16(buf[0])
	if totalLen == 0 {
		return nil, nil
	}
	if totalLen%2 != 0 {
		return nil, errors.New("getClientVersion error, wrong packet format")
	}
	index := uint16(0)
	buf = buf[1:]
	var res []uint16
	for index < totalLen {
		version := uint16(buf[index])
		version = version << 8
		version |= uint16(buf[index+1])
		index += 2
		res = append(res, version)
	}

	return res, nil
}
