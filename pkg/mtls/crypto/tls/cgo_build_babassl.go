// Actually we should use -I -I${SRCDIR}/darwin_Babassl_lib/include to include
// *.h of BabaSSL, but Babassl is not open source at present, so we use openssl
// header file for tmp

// +build BabaSSL

package tls


// #cgo darwin CFLAGS: -I${SRCDIR}/darwin_openssl_lib/include -Wno-deprecated-declarations
// #cgo darwin LDFLAGS: -L${SRCDIR}/darwin_BabaSSL_lib/lib -lssl -lcrypto
// #cgo linux CFLAGS: -I${SRCDIR}/linux_openssl_lib/include -Wno-deprecated-declarations
// #cgo linux LDFLAGS: -L${SRCDIR}/linux_BabaSSL_lib/lib ${SRCDIR}/linux_BabaSSL_lib/lib/libssl.a ${SRCDIR}/linux_BabaSSL_lib/lib/libcrypto.a -ldl
import "C"


// TLS13GmCipher define should in cgo_ciphersuites.go, for tls1.3+gm nor support for openssl
// define here for tmp
var TLS13GmCipher = []*Ciphersuites{
	{TLS_SM4_GCM_SM3, "TLS_SM4_GCM_SM3"},
	{TLS_SM4_CCM_SM3, "TLS_SM4_CCM_SM3"},
}