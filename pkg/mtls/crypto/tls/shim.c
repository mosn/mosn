#include <string.h>

#include <openssl/conf.h>

#include <openssl/bio.h>
#include <openssl/crypto.h>
#include <openssl/engine.h>
#include <openssl/err.h>
#include <openssl/evp.h>
#include <openssl/ssl.h>

#include "_cgo_export.h"

extern int go_init_locks();
extern void go_thread_locking_callback(int, int, const char*, int);
extern unsigned long go_thread_id_callback();
static int go_write_bio_puts(BIO *b, const char *str) {
	return go_write_bio_write(b, (char*)str, (int)strlen(str));
}


void X_BIO_set_data(BIO* bio, void* data) {
	BIO_set_data(bio, data);
}

void* X_BIO_get_data(BIO* bio) {
	return BIO_get_data(bio);
}

static int x_bio_create(BIO *b) {
	BIO_set_shutdown(b, 1);
	BIO_set_init(b, 1);
	BIO_set_data(b, NULL);
	BIO_clear_flags(b, ~0);
	return 1;
}

static int x_bio_free(BIO *b) {
	return 1;
}

static BIO_METHOD *writeBioMethod;
static BIO_METHOD *readBioMethod;

BIO_METHOD* BIO_s_readBio() { return readBioMethod; }
BIO_METHOD* BIO_s_writeBio() { return writeBioMethod; }

int x_bio_init_methods() {
	writeBioMethod = BIO_meth_new(BIO_TYPE_SOURCE_SINK, "Go Write BIO");
	if (!writeBioMethod) {
		return 1;
	}
	if (1 != BIO_meth_set_write(writeBioMethod,
				(int (*)(BIO *, const char *, int))go_write_bio_write)) {
		return 2;
	}
	if (1 != BIO_meth_set_puts(writeBioMethod, go_write_bio_puts)) {
		return 3;
	}
	if (1 != BIO_meth_set_ctrl(writeBioMethod, go_write_bio_ctrl)) {
		return 4;
	}
	if (1 != BIO_meth_set_create(writeBioMethod, x_bio_create)) {
		return 5;
	}
	if (1 != BIO_meth_set_destroy(writeBioMethod, x_bio_free)) {
		return 6;
	}

	readBioMethod = BIO_meth_new(BIO_TYPE_SOURCE_SINK, "Go Read BIO");
	if (!readBioMethod) {
		return 7;
	}
	if (1 != BIO_meth_set_read(readBioMethod, go_read_bio_read)) {
		return 8;
	}
	if (1 != BIO_meth_set_ctrl(readBioMethod, go_read_bio_ctrl)) {
		return 9;
	}
	if (1 != BIO_meth_set_create(readBioMethod, x_bio_create)) {
		return 10;
	}
	if (1 != BIO_meth_set_destroy(readBioMethod, x_bio_free)) {
		return 11;
	}

	return 0;
}

int X_shim_init() {
	int rc = 0;

	OPENSSL_config(NULL);
	ENGINE_load_builtin_engines();
	SSL_load_error_strings();
	SSL_library_init();
	OpenSSL_add_all_algorithms();
	//
	// Set up OPENSSL thread safety callbacks.
	rc = go_init_locks();
	if (rc != 0) {
		return rc;
	}
	CRYPTO_set_locking_callback(go_thread_locking_callback);
	CRYPTO_set_id_callback(go_thread_id_callback);

	rc = x_bio_init_methods();
	if (rc != 0) {
		return rc;
	}

	return 0;
}

int X_BIO_get_flags(BIO *b) {
	return BIO_get_flags(b);
}

void X_BIO_set_flags(BIO *b, int flags) {
	return BIO_set_flags(b, flags);
}

void X_BIO_clear_flags(BIO *b, int flags) {
	BIO_clear_flags(b, flags);
}

int X_BIO_read(BIO *b, void *buf, int len) {
	return BIO_read(b, buf, len);
}

int X_BIO_write(BIO *b, const void *buf, int len) {
	return BIO_write(b, buf, len);
}

BIO *X_BIO_new_write_bio() {
	return BIO_new(BIO_s_writeBio());
}

BIO *X_BIO_new_read_bio() {
	return BIO_new(BIO_s_readBio());
}

void ssl_ctx_set_hostname_cert_verify_cb(SSL_CTX *ctx, void *arg)
{
	SSL_CTX_set_cert_verify_callback(ctx, clientHostNameCertVerifyCallback,
			                         arg);
}

int ssl_cert_verify_call_back_always_success(int i, X509_STORE_CTX *x)
{
	return 1;
}

void ssl_set_cert_verify_require_peer_cert(SSL *ssl)
{
	SSL_set_verify(ssl, SSL_VERIFY_PEER|SSL_VERIFY_FAIL_IF_NO_PEER_CERT,
		ssl_cert_verify_call_back_always_success);
}

void ssl_ctx_set_cert_verify_require_peer_cert(SSL_CTX *ctx)
{
	SSL_CTX_set_verify(ctx, SSL_VERIFY_PEER|SSL_VERIFY_FAIL_IF_NO_PEER_CERT,
		ssl_cert_verify_call_back_always_success);
}


void ssl_ctx_set_client_hello_cb_GetConfigForClient(SSL_CTX *ctx, void *arg) {
	SSL_CTX_set_client_hello_cb(ctx,
	                            serverClientHelloCallBackForGetConfigForClient, arg);
}

void ssl_ctx_set_cert_verify_callback_serverVerifyBackForVerifyPeerCertificate(SSL_CTX *ctx, void *arg)
{
	SSL_CTX_set_cert_verify_callback(ctx,
	                                 serverVerifyBackForVerifyPeerCertificate, arg);
}

void ssl_ctx_set_cert_verify_callback_serverCertVerifyCallBack(SSL_CTX *ctx, void *arg)
{
	SSL_CTX_set_cert_verify_callback(ctx,
	                                 severCertVerifyCallBack, arg);
}

void ssl_ctx_set_cert_verify_callback_clientCertVerifyCallBack(SSL_CTX *ctx, void *arg)
{
	SSL_CTX_set_cert_verify_callback(ctx,
	                                 clientCertVerifyCallBack, arg);
}

void ssl_ctx_set_defaultClientServerNameCallBack(SSL_CTX *ctx)
{
	SSL_CTX_set_tlsext_servername_callback(ctx, defaultClientServerNameCallBack);
}

uint16_t packet_peek_2byte_len(const unsigned char* str)
{
	uint16_t res;

	res = ((uint16_t)(*str)) << 8;
	res |= *(str + 1);

	return res;
}