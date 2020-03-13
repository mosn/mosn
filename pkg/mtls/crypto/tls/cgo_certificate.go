package tls

/*
#include "shim.h"

static X509 *SSL_read_Certificate_file(const char *cert_file, int type)
{
    int j;
    BIO *in;
    X509 *x = NULL;

    in = BIO_new(BIO_s_file());
    if (in == NULL) {
        goto end;
    }

    if (BIO_read_filename(in, cert_file) <= 0) {
        goto end;
    }
    if (type == SSL_FILETYPE_ASN1) {
        j = ERR_R_ASN1_LIB;
        x = d2i_X509_bio(in, NULL);
    } else if (type == SSL_FILETYPE_PEM) {
        j = ERR_R_PEM_LIB;
        x = PEM_read_bio_X509(in, NULL, NULL, NULL);
    } else {
        goto end;
    }

    if (x == NULL) {
        goto end;
    }
end:
    BIO_free(in);
    return x;
}

static EVP_PKEY *SSL_read_PrivateKey_file(const char *pkey_file, int type)
{
    int j = 0;
    BIO *in;
    EVP_PKEY *pkey = NULL;

    in = BIO_new(BIO_s_file());
    if (in == NULL) {
        goto end;
    }

    if (BIO_read_filename(in, pkey_file) <= 0) {
        goto end;
    }
    if (type == SSL_FILETYPE_PEM) {
        j = ERR_R_PEM_LIB;
        pkey = PEM_read_bio_PrivateKey(in, NULL, NULL, NULL);
    } else if (type == SSL_FILETYPE_ASN1) {
        j = ERR_R_ASN1_LIB;
        pkey = d2i_PrivateKey_bio(in, NULL);
    } else {
        goto end;
    }
    if (pkey == NULL) {
        goto end;
    }
 end:
    BIO_free(in);
    return pkey;
}

static X509 *BabaSSL_read_Certificate_bytes(void *buf, int len)
{
    X509 *x = NULL;
    BIO* bio;

    bio = BIO_new_mem_buf(buf, len);
    if (bio == NULL) {
        goto end;
    }

    x = PEM_read_bio_X509_AUX(bio, NULL, NULL, NULL);
    if (x == NULL) {
        printf("err in PEM_read_bio_X509_AUX ");
    }

end:
    BIO_free(bio);
    return x;
}

static STACK_OF(X509) *BabaSSL_read_Certificate_chain_bytes(void *buf, int len)
{
	int i;
	STACK_OF(X509_INFO) *xis = NULL;
	X509_INFO *xi;
	STACK_OF(X509) *sk = NULL;
    BIO* bio;

    bio = BIO_new_mem_buf(buf, len);
    if (bio == NULL) {
        goto end;
	}

	sk = sk_X509_new_null();
	if (sk == NULL) {
        goto end;
	}

	xis = PEM_X509_INFO_read_bio(bio, NULL, NULL, NULL);

	for (i = 0; i < sk_X509_INFO_num(xis); i++) {
        xi = sk_X509_INFO_value(xis, i);
        if (xi->x509 != NULL && sk != NULL) {
            if (!sk_X509_push(sk, xi->x509))
                goto end;
            xi->x509 = NULL;
        }
    }

end:
	BIO_free(bio);
	sk_X509_INFO_pop_free(xis, X509_INFO_free);
    return sk;
}

static EVP_PKEY *BabaSSL_read_PrivateKey_bytes(void *buf, int len)
{
    EVP_PKEY *pkey = NULL;
    BIO* bio;

    bio = BIO_new_mem_buf(buf, len);
    if (bio == NULL) {
        goto end;
    }

    pkey = PEM_read_bio_PrivateKey(bio, NULL, NULL, NULL);
    if (pkey == NULL) {
        printf("err in PEM_read_bio_PrivateKey ");
    }

end:
    BIO_free(bio);
    return pkey;

}

static int translate_x509_to_raw_byte(X509 *x, void *buf)
{
	int cert_len = 0;

	cert_len = i2d_X509_AUX(x, (unsigned char **)(&buf));

	return cert_len;
}

static int SSL_CTX_set0_chain_cgo(SSL_CTX *ctx, STACK_OF(X509) *chain)
{
	return SSL_CTX_set0_chain(ctx, chain);
}

static int SSL_CTX_set1_chain_cgo(SSL_CTX *ctx, STACK_OF(X509) *chain)
{
	return SSL_CTX_set1_chain(ctx, chain);
}

static int SSL_CTX_build_cert_chain_cgo(SSL_CTX *ctx, int chflags)
{
    return SSL_CTX_build_cert_chain(ctx, chflags);
}

static int SSL_set0_chain_cgo(SSL *ssl, STACK_OF(X509) *chain)
{
	return SSL_set0_chain(ssl, chain);
}

static int SSL_set1_chain_cgo(SSL *ssl, STACK_OF(X509) *chain)
{
	return SSL_set1_chain(ssl, chain);
}

static int SSL_build_cert_chain_cgo(SSL *ssl, int chflags)
{
    return SSL_build_cert_chain(ssl, chflags);
}
*/
import "C"
import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"net"
	"unsafe"
)

//SslReadCertificateFile read cetificate from file
//if read processing faild, return error
func SslReadCertificateFile(certFile string) (*C.X509, error) {
	x := C.SSL_read_Certificate_file(C.CString(certFile), C.SSL_FILETYPE_PEM)
	if x == nil {
		return nil, errors.New("read certificat error")
	}

	return x, nil
}

// SslReadPrivateKeyFile read pkey from file
// if read processing faild, return error
func SslReadPrivateKeyFile(pkeyFile string) (*C.EVP_PKEY, error) {
	pkey := C.SSL_read_PrivateKey_file(C.CString(pkeyFile), C.SSL_FILETYPE_PEM)

	if pkey == nil {
		return nil, errors.New("read certificat error")
	}

	return pkey, nil
}

// TranslateRawByteToSslX509 translate []byte from buf to openssl X509*
// if translate processing faild, return error
func TranslateRawByteToSslX509(buf []byte) (*C.X509, error) {
	x := C.BabaSSL_read_Certificate_bytes(unsafe.Pointer(&buf[0]), C.int(len(buf)))
	if x == nil {
		return nil, errors.New("TranslateRawByteToSslX509 error")
	}

	return x, nil
}

// TranslateRawByteToSslX509Chain translate []byte from buf to openssl STACK_OF(X509)*
// if translate processing faild, return error
func TranslateRawByteToSslX509Chain(buf []byte) (*C.struct_stack_st_X509, error) {
	sk := C.BabaSSL_read_Certificate_chain_bytes(unsafe.Pointer(&buf[0]), C.int(len(buf)))
	if sk == nil {
		return nil, errors.New("TranslateRawByteToSslX509Chain error")
	}

	if int(C.sk_X509_num(sk)) <= 1 {
		//if only have one cert, need not build cert chain
		return nil, nil
	}

	return sk, nil
}

// TranslateRawByteToSslPrivateKey translate []byte from buf to openssl EVP_PKEY*
// if translate processing faild, return error
func TranslateRawByteToSslPrivateKey(buf []byte) (*C.EVP_PKEY, error) {
	pkey := C.BabaSSL_read_PrivateKey_bytes(unsafe.Pointer(&buf[0]), C.int(len(buf)))

	if pkey == nil {
		return nil, errors.New("TranslateRawByteToSslPrivateKey error")
	}

	return pkey, nil
}

// SetVerifyCertsIntoCtx add trusted caCert to openssl ssl_ctx*
// if set processing faild, return error
func SetVerifyCertsIntoCtx(ctx *C.SSL_CTX, caCerts []*C.X509) error {
	if caCerts != nil {
		vrf := C.X509_STORE_new()
		if vrf == nil {
			return errors.New("SetVerifyCertsIntoCtx error, X509_STORE_new error")
		}
		for _, cert := range caCerts {
			ret := C.X509_STORE_add_cert(vrf, cert)
			if int(ret) <= 0 {
				return errors.New("SetVerifyCertsIntoCtx error, X509_STORE_add_cert error")
			}
		}
		//C.SSL_CTX_set1_verify_cert_store(ctx, vfy)
		C.SSL_CTX_ctrl(ctx, C.SSL_CTRL_SET_VERIFY_CERT_STORE, 1, unsafe.Pointer(vrf))
	}
	return nil
}

// SetVerifyCertsIntoSsl add trusted caCert to openssl ssl*
// if set processing faild, return error
func SetVerifyCertsIntoSsl(ssl *C.SSL, caCerts []*C.X509) error {
	if caCerts != nil {
		vrf := C.X509_STORE_new()
		if vrf == nil {
			return errors.New("X509_STORE_new error")
		}
		for _, cert := range caCerts {
			ret := C.X509_STORE_add_cert(vrf, cert)
			if int(ret) <= 0 {
				return errors.New("X509_STORE_add_cert error")
			}
		}
		//SSL_set1_verify_cert_store(ssl, st)
		C.SSL_ctrl(ssl, C.SSL_CTRL_SET_VERIFY_CERT_STORE, 1, unsafe.Pointer(vrf))
	}
	return nil
}

// TranslateGoCertToSslX509 translate x509.Certificate to openssl X509*
// if translate processing faild, return error
func TranslateGoCertToSslX509(cert *x509.Certificate) (*C.X509, error) {
	var cX509 *C.X509
	var err error
	if cert != nil {
		certByte := cert.Raw
		buf := bytes.NewBufferString("")
		pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: certByte})
		cX509, err = TranslateRawByteToSslX509(buf.Bytes())
		if err != nil {
			return nil, err
		}

	}
	return cX509, nil
}

// TranslateGoCertsToSslX509s translate []x509.Certificate to []X509*
// if translate processing faild, return error
func TranslateGoCertsToSslX509s(certs []*x509.Certificate) ([]*C.X509, error) {
	var cX509s []*C.X509
	if certs != nil {
		for _, cert := range certs {
			certByte := cert.Raw
			buf := bytes.NewBufferString("")
			pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: certByte})
			cX509, err := TranslateRawByteToSslX509(buf.Bytes())
			if err != nil {
				return nil, err
			}
			cX509s = append(cX509s, cX509)
		}

	}
	return cX509s, nil
}

// TranslateSslX509ToRawByte translate openssl X509* to pem encode byte
// if translate processing faild, return error
func TranslateSslX509ToRawByte(x *C.X509) ([]byte, error) {
	buf := make([]byte, 102400)
	certLen := C.translate_x509_to_raw_byte(x, unsafe.Pointer(&buf[0]))
	if int(certLen) <= 0 {
		return nil, errors.New("TranslateSslX509ToRawByte error")
	}

	certByte := buf[:certLen]
	return certByte, nil
}

// TranslateSslX509ToGoCert translate openssl X509* to go *x509.Certificate
// if translate processing faild, return error
func TranslateSslX509ToGoCert(x *C.X509) (*x509.Certificate, error) {
	rawByte, err := TranslateSslX509ToRawByte(x)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(rawByte)
}

// TranslateSslX509StackToGoCerts translate openssl STACK_OF(X509)* to go []*x509.Certificate
// if translate processing faild, return error
func TranslateSslX509StackToGoCerts(sk *C.struct_stack_st_X509) ([]*x509.Certificate, error) {
	var res []*x509.Certificate
	if sk != nil {
		certNum := int(C.sk_X509_num(sk))
		if certNum <= 0 {
			return nil, errors.New("getPeerCertsAsGoRawByte error, peer don't provide cert")
		}

		for i := 0; i < certNum; i++ {
			certSsl := C.sk_X509_value(sk, C.int(i))
			certGo, err := TranslateSslX509ToGoCert(certSsl)
			if err != nil {
				return nil, errors.New("getPeerCertsAsGoRawByte error, tran cert error")
			}
			res = append(res, certGo)
		}
	}

	return res, nil
}

// TranslateSslX509StackToRawBytes translate openssl STACK_OF(X509)* to pem encode rawbytes
// if translate processing faild, return error
func TranslateSslX509StackToRawBytes(sk *C.struct_stack_st_X509) ([][]byte, error) {
	var res [][]byte
	if sk != nil {
		certNum := int(C.sk_X509_num(sk))
		if certNum <= 0 {
			return nil, errors.New("getPeerCertsAsGoRawByte error, peer don't provide cert")
		}

		for i := 0; i < certNum; i++ {
			certX509 := C.sk_X509_value(sk, C.int(i))
			certRawByte, err := TranslateSslX509ToRawByte(certX509)
			if err != nil {
				return nil, err
			}
			res = append(res, certRawByte)
		}
	}
	return res, nil
}

// clientHostNameCertVerifyCallback would be called in openssl state machine(ctx->app_verify_callback)
// Its purpose is to verify that server name is matched Certificate
//export clientHostNameCertVerifyCallback
func clientHostNameCertVerifyCallback(ctx *C.X509_STORE_CTX, arg unsafe.Pointer) C.int {
	h := *(*string)(arg)
	cert := C.X509_STORE_CTX_get0_cert(ctx)
	if cert == nil {
		if BabasslPrintTraceTag.IsOpen() {
			print("clientHostNameCertVerifyCallback error, don't have cert")
		}
		return -1
	}

	hCchar := C.CString(h)
	//hC_uchar := (*C.uchar)(unsafe.Pointer(hCchar))
	hLenCulong := C.ulong(len(h))
	candidateIP := h
	if len(h) >= 3 && h[0] == '[' && h[len(h)-1] == ']' {
		candidateIP = h[1 : len(h)-1]
	}
	if ip := net.ParseIP(candidateIP); ip != nil {
		ret := C.X509_check_ip_asc(cert, hCchar, 0)
		if int(ret) <= 0 {
			if BabasslPrintTraceTag.IsOpen() {
				print("X509_check_ip error")
			}
			return -1
		}
		return 1
	}

	ret := C.X509_check_host(cert, hCchar, hLenCulong, 0, nil)
	if int(ret) <= 0 {
		if BabasslPrintTraceTag.IsOpen() {
			print("X509_check_host error")
		}
		return -1
	}

	return 1
}

func getPeerCertsAsX509Arr(s *C.SSL) ([]*x509.Certificate, error) {
	certChain := C.SSL_get_peer_cert_chain(s)
	return TranslateSslX509StackToGoCerts(certChain)
}

func getPeerCertsAsGoRawByte(s *C.SSL) ([][]byte, error) {
	//var tmp *C.struct_stack_st_X509
	certChain := C.SSL_get_peer_cert_chain(s)
	return TranslateSslX509StackToRawBytes(certChain)
}

func tranGoRawByteCertsToSslCertChain(certs [][]byte) (*C.struct_stack_st_X509, error) {
	var sk *C.struct_stack_st_X509

	sk = C.sk_X509_new_null()
	if sk == nil {
		return nil, errors.New(`tranGoRawByteCertsToSslCertChain error,
		    can't generate sk_X509_`)
	}

	for i := len(certs) - 1; i >= 0; i-- {
		certRawByte := certs[i]
		buf := bytes.NewBufferString("")
		pem.Encode(buf, &pem.Block{Type: "CERTIFICATE", Bytes: certRawByte})
		cX509, err := TranslateRawByteToSslX509(buf.Bytes())
		if err != nil {
			return nil, err
		}

		ret := int(C.sk_X509_push(sk, cX509))
		if ret <= 0 {
			return nil, errors.New(`tranGoRawByteCertsToSslCertChain error,
			    sk_X509_push error`)
		}
	}
	return sk, nil
}

func setSslCertAndPkeyToSslCtx(sslCtx *C.SSL_CTX, sslCert *SslCertificate) error {
	var ret int
	ret = int(C.SSL_CTX_use_certificate(sslCtx, sslCert.Cert))
	if ret <= 0 {
		return errors.New("serverSslCtxInit error, SSL_CTX_use_certificate error")
	}

	ret = int(C.SSL_CTX_use_PrivateKey(sslCtx, sslCert.Pkey))
	if ret <= 0 {
		return errors.New("serverSslCtxInit error, SSL_CTX_use_PrivateKey error")
	}

	if sslCert.CertChain != nil {
		ret = int(C.SSL_CTX_set1_chain_cgo(sslCtx, sslCert.CertChain))
		if ret <= 0 {
			return errors.New("serverSslCtxInit error, SSL_CTX_set0_chain error")
		}

		ret = int(C.SSL_CTX_build_cert_chain_cgo(sslCtx, C.SSL_BUILD_CHAIN_FLAG_CHECK))
		if ret <= 0 {
			return errors.New("serverSslCtxInit error, SSL_CTX_build_cert_chain error")
		}
	}

	return nil
}

func setSslCertAndPkeyToSsl(ssl *C.SSL, sslCert *SslCertificate) error {
	var ret int
	ret = int(C.SSL_use_certificate(ssl, sslCert.Cert))
	if ret <= 0 {
		return errors.New("serverSslCtxInit error, SSL_CTX_use_certificate error")
	}

	ret = int(C.SSL_use_PrivateKey(ssl, sslCert.Pkey))
	if ret <= 0 {
		return errors.New("serverSslCtxInit error, SSL_CTX_use_PrivateKey error")
	}

	if sslCert.CertChain != nil {
		ret = int(C.SSL_set1_chain_cgo(ssl, sslCert.CertChain))
		if ret <= 0 {
			return errors.New("serverSslCtxInit error, SSL_CTX_set0_chain error")
		}

		ret = int(C.SSL_build_cert_chain_cgo(ssl, C.SSL_BUILD_CHAIN_FLAG_CHECK))
		if ret <= 0 {
			return errors.New("serverSslCtxInit error, SSL_CTX_build_cert_chain error")
		}
	}

	return nil
}

//export severCertVerifyCallBack
func severCertVerifyCallBack(xs *C.X509_STORE_CTX, arg unsafe.Pointer) C.int {
	conf, ok := cgoPointerRestore(arg).(*Config)
	cgoPointerUnref(arg)
	if !ok {
		return -1
	}

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

	var chains [][]*x509.Certificate
	if conf.ClientAuth >= VerifyClientCertIfGiven {
		opts := x509.VerifyOptions{
			Roots:         conf.ClientCAs,
			CurrentTime:   conf.time(),
			Intermediates: x509.NewCertPool(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		for _, cert := range peerCerts[1:] {
			opts.Intermediates.AddCert(cert)
		}

		chains, err = peerCerts[0].Verify(opts)
		if err != nil {
			return -1
		}
	}

	if conf.VerifyPeerCertificate != nil {
		if err := conf.VerifyPeerCertificate(peerCertsRaw, chains); err != nil {
			return -1
		}
	}
	return 1
}

//export clientCertVerifyCallBack
func clientCertVerifyCallBack(xs *C.X509_STORE_CTX, arg unsafe.Pointer) C.int {
	conf, ok := cgoPointerRestore(arg).(*Config)
	cgoPointerUnref(arg)
	if !ok {
		return -1
	}

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

	var chains [][]*x509.Certificate
	if !conf.InsecureSkipVerify {
		opts := x509.VerifyOptions{
			Roots:         conf.ClientCAs,
			CurrentTime:   conf.time(),
			Intermediates: x509.NewCertPool(),
			KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}

		for _, cert := range peerCerts[1:] {
			opts.Intermediates.AddCert(cert)
		}

		chains, err = peerCerts[0].Verify(opts)
		if err != nil {
			return -1
		}
	}

	if conf.VerifyPeerCertificate != nil {
		if err := conf.VerifyPeerCertificate(peerCertsRaw, chains); err != nil {
			return -1
		}
	}
	return 1
}
