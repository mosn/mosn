package tls

/*
#include "shim.h"

static int client_ssl_servername_cb(SSL *s, int *ad, void *arg)
{
    return SSL_TLSEXT_ERR_OK;
}

static long SSL_CTX_set_tlsext_servername_callback_cgo(SSL_CTX *ctx, void *cb)
{
	return SSL_CTX_callback_ctrl(ctx,SSL_CTRL_SET_TLSEXT_SERVERNAME_CB,
                          (void (*)(void))cb);
}

static long SSL_CTX_set_tlsext_servername_arg_cgo(SSL_CTX *ctx, void *arg)
{
	return SSL_CTX_ctrl(ctx,SSL_CTRL_SET_TLSEXT_SERVERNAME_ARG,0,arg);
}

static long SSL_set_tlsext_host_name_cgo(SSL *s, char *name)
{
	return SSL_ctrl(s,SSL_CTRL_SET_TLSEXT_HOSTNAME,TLSEXT_NAMETYPE_host_name,
		     (void *)name);
}

*/
import "C"
import (
	"errors"
	"net"
	"unsafe"
)

//export defaultClientServerNameCallBack
func defaultClientServerNameCallBack(ssl *C.SSL, i *C.int, arg unsafe.Pointer) C.int {
	return 1
}

func clientSetSslCtxServerNameCallBack(ctx *C.SSL_CTX, arg string) {
	C.ssl_ctx_set_defaultClientServerNameCallBack(ctx)
	C.SSL_CTX_set_tlsext_servername_arg_cgo(ctx, unsafe.Pointer(&arg))
}

func clientSslSetServerName(ssl *C.SSL, serverName string) error {
	ret := C.SSL_set_tlsext_host_name_cgo(ssl, C.CString(serverName))

	if int(ret) <= 0 {
		return errors.New("SSL_set_tlsext_host_name error")
	}

	return nil
}

func checkServerNameMatchCertificate(ssl *C.SSL, h string) error {
	cert := C.SSL_get_peer_certificate(ssl)
	if cert == nil {
		return errors.New("checkServerNameMatchCertificate error, no peer cert")
	}
	hCchar := C.CString(h)
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
			return errors.New("checkServerNameMatchCertificate error")
		}
		return nil
	}

	ret := C.X509_check_host(cert, hCchar, hLenCulong, C.X509_CHECK_FLAG_NEVER_CHECK_SUBJECT, nil)
	if int(ret) <= 0 {
		if BabasslPrintTraceTag.IsOpen() {
			print("X509_check_host error")
		}
		return errors.New("checkServerNameMatchCertificate error")
	}
	return nil
}
