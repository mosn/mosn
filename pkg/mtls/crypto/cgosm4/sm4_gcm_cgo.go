// +build BabaSSL

package cgosm4

// #cgo darwin CFLAGS: -I/usr/local/BabaSSL/darwin_BabaSSL_lib/include -Wno-deprecated-declarations
// #cgo darwin LDFLAGS: -L/usr/local/BabaSSL/darwin_BabaSSL_lib/lib -lssl -lcrypto
// #cgo linux,!arm,!arm64 CFLAGS: -I/usr/local/BabaSSL/linux_BabaSSL_lib/include -Wno-deprecated-declarations
// #cgo linux,!arm,!arm64 LDFLAGS: -L/usr/local/BabaSSL/linux_BabaSSL_lib/lib -lssl -lcrypto -ldl -lpthread
// #cgo linux,arm64 CFLAGS: -I/usr/local/BabaSSL/linux_BabaSSL_arm_lib/include -Wno-deprecated-declarations
// #cgo linux,arm64 LDFLAGS: -L/usr/local/BabaSSL/linux_BabaSSL_arm_lib/lib -lssl -lcrypto -ldl -lpthread
/*
#include <openssl/evp.h>

int sm4_gcm_encrypt(unsigned char *plaintext, int plaintext_len,
                	unsigned char *aad, int aad_len,
                	unsigned char *key,
                	unsigned char *iv, int iv_len,
                	unsigned char *ciphertext,
               	 	unsigned char *tag)
{
    EVP_CIPHER_CTX *ctx;

    int len;

    int ciphertext_len;

    if(!(ctx = EVP_CIPHER_CTX_new()))
        return -1;

    if(1 != EVP_EncryptInit_ex(ctx, EVP_sm4_gcm(), NULL, NULL, NULL))
        return -1;

    if(1 != EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, iv_len, NULL))
        return -1;

    if(1 != EVP_EncryptInit_ex(ctx, NULL, NULL, key, iv))
        return -1;

    if(1 != EVP_EncryptUpdate(ctx, NULL, &len, aad, aad_len))
        return -1;

    if(1 != EVP_EncryptUpdate(ctx, ciphertext, &len, plaintext, plaintext_len))
        return -1;
    ciphertext_len = len;

    if(1 != EVP_EncryptFinal_ex(ctx, ciphertext + len, &len))
        return -1;
    ciphertext_len += len;

    if(1 != EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_GET_TAG, 16, tag))
		return -1;

    EVP_CIPHER_CTX_free(ctx);

    return ciphertext_len;
}

int sm4_gcm_decrypt(unsigned char *ciphertext, int ciphertext_len,
                	unsigned char *aad, int aad_len,
                	unsigned char *tag,
               		unsigned char *key,
                	unsigned char *iv, int iv_len,
                	unsigned char *plaintext)
{
    EVP_CIPHER_CTX *ctx;
    int len;
    int plaintext_len;
    int ret;

    if(!(ctx = EVP_CIPHER_CTX_new()))
        return -1;

    if(!EVP_DecryptInit_ex(ctx, EVP_sm4_gcm(), NULL, NULL, NULL))
        return -1;

    if(!EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_IVLEN, iv_len, NULL))
        return -1;

    if(!EVP_DecryptInit_ex(ctx, NULL, NULL, key, iv))
        return -1;

    if(!EVP_DecryptUpdate(ctx, NULL, &len, aad, aad_len))
        return -1;

    if(!EVP_DecryptUpdate(ctx, plaintext, &len, ciphertext, ciphertext_len))
        return -1;
    plaintext_len = len;

    if(!EVP_CIPHER_CTX_ctrl(ctx, EVP_CTRL_GCM_SET_TAG, 16, tag))
        return -1;

    ret = EVP_DecryptFinal_ex(ctx, plaintext + len, &len);

    EVP_CIPHER_CTX_free(ctx);

    if(ret > 0) {
        plaintext_len += len;
        return plaintext_len;
    } else {
        return -1;
    }
}

*/
import "C"
import (
	"crypto/cipher"
	"errors"
	"unsafe"
)

const (
	gcmBlockSize         = 16
	gcmTagSize           = 16
	gcmMinimumTagSize    = 12 // NIST SP 800-38D recommends tags with 12 or more bytes.
	gcmStandardNonceSize = 12
)

// Sm4Gcm is basic type of sm4, which implement all cipher.AEAD interface
type Sm4Gcm struct {
	key []byte
}

// NewGCM return a basic Sm4Gcm struct
func NewGCM(k []byte) cipher.AEAD {
	return &Sm4Gcm{
		key: k,
	}
}

// NonceSize required by cipher.AEAD interface
// sm4-gcm nonce size is 12
func (sg *Sm4Gcm) NonceSize() int {
	return gcmStandardNonceSize
}

// Overhead required by cipher.AEAD interface, always equal tag size
// sm4-gcm tag size is 12
func (sg *Sm4Gcm) Overhead() int {
	return gcmTagSize
}

func (sg *Sm4Gcm) tagSize() int {
	return gcmTagSize
}

// Seal required by cipher.AEAD interface
func (sg *Sm4Gcm) Seal(dst, nonce, plaintext, additionalData []byte) []byte {
	if len(nonce) != sg.NonceSize() {
		panic("crypto/cipher: incorrect nonce length given to GCM")
	}

	if uint64(len(plaintext)) > ((1<<32)-2)*uint64(gcmBlockSize) {
		panic("crypto/cipher: message too large for GCM")
	}

	ret, out := sliceForAppend(dst, len(plaintext)+sg.tagSize())

	// sm4 key size should euqal block size, the value is 16
	if len(sg.key) != gcmBlockSize {
		panic("crypto/cipher: invalid buffer overlap")
	}

	encLen := C.sm4_gcm_encrypt((*C.uchar)(unsafe.Pointer(&plaintext[0])), C.int(len(plaintext)),
		(*C.uchar)(unsafe.Pointer(&additionalData[0])), C.int(len(additionalData)),
		(*C.uchar)(unsafe.Pointer(&sg.key[0])),
		(*C.uchar)(unsafe.Pointer(&nonce[0])), C.int(len(nonce)),
		(*C.uchar)(unsafe.Pointer(&out[0])),
		(*C.uchar)(unsafe.Pointer(&out[len(plaintext)])))

	if int(encLen) != len(plaintext) {
		panic("error happen in sm4_gcm_encrypt")
	}

	return ret
}

var errOpen = errors.New("cipher: message authentication failed")

// Open required by cipher.AEAD interface
func (sg *Sm4Gcm) Open(dst, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	if len(nonce) != sg.NonceSize() {
		panic("crypto/cipher: incorrect nonce length given to GCM")
	}
	// Sanity check to prevent the authentication from always succeeding if an implementation
	// leaves tagSize uninitialized, for example.
	if sg.tagSize() < gcmMinimumTagSize {
		panic("crypto/cipher: incorrect GCM tag size")
	}

	if len(ciphertext) < sg.tagSize() {
		return nil, errOpen
	}
	if uint64(len(ciphertext)) > ((1<<32)-2)*uint64(gcmBlockSize)+uint64(sg.tagSize()) {
		return nil, errOpen
	}

	tag := ciphertext[len(ciphertext)-sg.tagSize():]
	ciphertext = ciphertext[:len(ciphertext)-sg.tagSize()]
	ret := make([]byte, len(ciphertext), len(ciphertext))

	decRes := C.sm4_gcm_decrypt((*C.uchar)(unsafe.Pointer(&ciphertext[0])), C.int(len(ciphertext)),
		(*C.uchar)(unsafe.Pointer(&additionalData[0])), C.int(len(additionalData)),
		(*C.uchar)(unsafe.Pointer(&tag[0])),
		(*C.uchar)(unsafe.Pointer(&sg.key[0])),
		(*C.uchar)(unsafe.Pointer(&nonce[0])), C.int(len(nonce)),
		(*C.uchar)(unsafe.Pointer(&ret[0])))

	if int(decRes) < 0 || int(decRes) != len(ciphertext) {
		return nil, errOpen
	}

	return ret, nil
}

func sliceForAppend(in []byte, n int) (head, tail []byte) {
	if total := len(in) + n; cap(in) >= total {
		head = in[:total]
	} else {
		head = make([]byte, total)
		copy(head, in)
	}
	tail = head[len(in):]
	return
}
