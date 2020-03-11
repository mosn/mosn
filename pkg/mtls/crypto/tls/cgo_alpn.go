package tls

/*
#include "shim.h"

static int next_protos_parse_to_go_buf(const char *in, void *buf)
{
    size_t len;
    unsigned char *out;
    size_t i, start = 0;
	size_t skipped = 0;
	int outlen;

    len = strlen(in);
    if (len == 0 || len >= 65535)
        return 0;

    out = OPENSSL_malloc(len + 1);
    for (i = 0; i <= len; ++i) {
        if (i == len || in[i] == ',') {
            if (i == start) {
                ++start;
                ++skipped;
                continue;
            }
            if (i - start > 255) {
                OPENSSL_free(out);
                return 0;
            }
            out[start-skipped] = (unsigned char)(i - start);
            start = i + 1;
        } else {
            out[i + 1 - skipped] = in[i];
        }
    }

    if (len <= skipped) {
        OPENSSL_free(out);
        return 0;
    }

	outlen = len + 1 - skipped;
	memcpy(buf, out, outlen);
	OPENSSL_free(out);
    return outlen;
}

typedef struct tlsextalpnctx_st {
    unsigned char *data;
    unsigned int len;
} tlsextalpnctx;

int alpn_cb(SSL *s, const unsigned char **out, unsigned char *outlen,
                   const unsigned char *in, unsigned int inlen, void *arg)
{
    tlsextalpnctx *alpn_ctx = arg;
    if (SSL_select_next_proto
        ((unsigned char **)out, outlen, alpn_ctx->data, alpn_ctx->len, in,
         inlen) != OPENSSL_NPN_NEGOTIATED) {
        return SSL_TLSEXT_ERR_NOACK;
    }
    return SSL_TLSEXT_ERR_OK;
}

//The call back function alpn_cb function can not be set into ctx direct
//in go code, because their complie unit are nor same
void set_alpn_cb_to_ctx(SSL_CTX *ctx, void *args)
{
	SSL_CTX_set_alpn_select_cb(ctx, alpn_cb, args);
}

static unsigned int get_ssl_alpn_select(SSL *ssl, void *buf)
{
	const unsigned char *alpn;
	unsigned int len;
	SSL_get0_alpn_selected(ssl, &alpn, &len);
	if (len == 0) {
		return 0;
	}
	memcpy(buf, alpn, len);
	return len;
}

void OPENSSL_free_cgo(void *addr)
{
	OPENSSL_free(addr);
}

*/
import "C"
import (
	"errors"
	"unsafe"
)

func clientSslCtxSetAlpnProtos(ctx *C.SSL_CTX, NextProtos []string) error {
	alpnProtos := ""
	for _, str := range NextProtos {
		alpnProtos += str
		alpnProtos += ","
	}
	buf := make([]byte, 256)
	len := int(C.next_protos_parse_to_go_buf(C.CString(alpnProtos), unsafe.Pointer(&buf[0])))
	if len <= 0 {
		return errors.New("client set alpn protos error")
	}
	alpn := C.CString(BytesToString(buf[:len]))
	alpnptr := (*C.uchar)(unsafe.Pointer(alpn))
	ret := C.SSL_CTX_set_alpn_protos(ctx, alpnptr, C.uint(len))
	if int(ret) != 0 {
		return errors.New("client set alpn protos error")
	}
	return nil
}

func serverSslCtxSetAlpnProtos(ctx *C.SSL_CTX, NextProtos []string) error {
	alpnProtos := ""
	for _, str := range NextProtos {
		alpnProtos += str
		alpnProtos += ","
	}

	buf := make([]byte, 256)
	len := int(C.next_protos_parse_to_go_buf(C.CString(alpnProtos), unsafe.Pointer(&buf[0])))
	if len <= 0 {
		return errors.New("client set alpn protos error")
	}
	alpn := C.CString(BytesToString(buf[:len]))
	alpnptr := (*C.uchar)(unsafe.Pointer(alpn))

	var alpnctx C.tlsextalpnctx
	alpnctx.data = alpnptr
	alpnctx.len = C.uint(len)

	C.set_alpn_cb_to_ctx(ctx, unsafe.Pointer(&alpnctx))
	return nil
}

func getSslAlpnNegotiated(ssl *C.SSL) (string, bool) {
	buf := make([]byte, 256)
	len := int(C.get_ssl_alpn_select(ssl, unsafe.Pointer(&buf[0])))
	if len == 0 {
		return "", false
	}
	buf = buf[:len]
	NegotiatedProtocol := BytesToString(buf)
	isMutual := true

	return NegotiatedProtocol, isMutual
}

func parseAlpnFromExtension(packet []byte) ([]string, error) {
	if len(packet) == 0 {
		return nil, nil
	}
	if len(packet) > 0 && len(packet) < 2 {
		return nil, errors.New("parseAlpnFromExtension error, error packet format")
	}
	totalLen, err := PacketPeek2byteToLen(packet)
	if err != nil {
		return nil, err
	}
	if totalLen == 0 {
		return nil, nil
	}
	packet = packet[2:]
	var res []string

	for {
		if len(packet) > 0 {
			subLen, err := PacketPeek1byteToLen(packet)
			if err != nil {
				return nil, err
			}
			if int(1+subLen) > len(packet) {
				return nil, errors.New("parseAlpnFromExtension error, error alpn format ")
			}
			protoByte := packet[1 : 1+subLen]
			protoString := BytesToString(protoByte)
			res = append(res, protoString)
			if int(1+subLen) == len(packet) {
				break
			} else {
				packet = packet[1+subLen:]
			}
		} else {
			break
		}
	}
	return res, nil
}
