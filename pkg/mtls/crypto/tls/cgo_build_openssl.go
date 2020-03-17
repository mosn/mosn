// +build !BabaSSL

package tls

// #cgo darwin CFLAGS: -I${SRCDIR}/darwin_openssl_lib/include -Wno-deprecated-declarations
// #cgo darwin LDFLAGS: -L${SRCDIR}/darwin_openssl_lib/lib -lssl -lcrypto
// #cgo linux,!arm,!arm64 CFLAGS: -I${SRCDIR}/linux_openssl_lib/include -Wno-deprecated-declarations
// #cgo linux,!arm,!arm64 LDFLAGS: -L${SRCDIR}/linux_openssl_lib/lib ${SRCDIR}/linux_openssl_lib/lib/libssl.a ${SRCDIR}/linux_openssl_lib/lib/libcrypto.a -ldl -lpthread
// #cgo linux,arm64 CFLAGS: -I${SRCDIR}/linux_openssl_arm_lib/include -Wno-deprecated-declarations
// #cgo linux,arm64 LDFLAGS: -L${SRCDIR}/linux_openssl_arm_lib/lib ${SRCDIR}/linux_openssl_arm_lib/lib/libssl.a ${SRCDIR}/linux_openssl_arm_lib/lib/libcrypto.a -ldl -lpthread
import "C"

// TLS13GmCipher is not support in openssl
var TLS13GmCipher = []*Ciphersuites{}
