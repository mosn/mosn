#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <signal.h>
#include <stdint.h>

#include <openssl/bio.h>
#include <openssl/crypto.h>
#include <openssl/dh.h>
#include <openssl/err.h>
#include <openssl/evp.h>
#include <openssl/hmac.h>
#include <openssl/pem.h>
#include <openssl/ssl.h>
#include <openssl/x509v3.h>
#include <openssl/ec.h>
#include <openssl/x509.h>

#ifndef SSL_MODE_RELEASE_BUFFERS
#define SSL_MODE_RELEASE_BUFFERS 0
#endif

#ifndef SSL_OP_NO_COMPRESSION
#define SSL_OP_NO_COMPRESSION 0
#endif

/* shim  methods */
extern int X_shim_init();

/* Library methods */
extern void X_OPENSSL_free(void *ref);
extern void *X_OPENSSL_malloc(size_t size);

/* SSL methods */
extern long X_SSL_set_options(SSL* ssl, long options);
extern long X_SSL_get_options(SSL* ssl);
extern long X_SSL_clear_options(SSL* ssl, long options);
extern long X_SSL_set_tlsext_host_name(SSL *ssl, const char *name);
extern const char * X_SSL_get_cipher_name(const SSL *ssl);
extern int X_SSL_session_reused(SSL *ssl);
extern int X_SSL_new_index();

extern const SSL_METHOD *X_SSLv23_method();
extern const SSL_METHOD *X_SSLv3_method();
extern const SSL_METHOD *X_TLSv1_method();
extern const SSL_METHOD *X_TLSv1_1_method();
extern const SSL_METHOD *X_TLSv1_2_method();

#if defined SSL_CTRL_SET_TLSEXT_HOSTNAME
extern int sni_cb(SSL *ssl_conn, int *ad, void *arg);
#endif
extern int X_SSL_verify_cb(int ok, X509_STORE_CTX* store);

/* SSL_CTX methods */
extern int X_SSL_CTX_new_index();
extern long X_SSL_CTX_set_options(SSL_CTX* ctx, long options);
extern long X_SSL_CTX_clear_options(SSL_CTX* ctx, long options);
extern long X_SSL_CTX_get_options(SSL_CTX* ctx);
extern long X_SSL_CTX_set_mode(SSL_CTX* ctx, long modes);
extern long X_SSL_CTX_get_mode(SSL_CTX* ctx);
extern long X_SSL_CTX_set_session_cache_mode(SSL_CTX* ctx, long modes);
extern long X_SSL_CTX_sess_set_cache_size(SSL_CTX* ctx, long t);
extern long X_SSL_CTX_sess_get_cache_size(SSL_CTX* ctx);
extern long X_SSL_CTX_set_timeout(SSL_CTX* ctx, long t);
extern long X_SSL_CTX_get_timeout(SSL_CTX* ctx);
extern long X_SSL_CTX_add_extra_chain_cert(SSL_CTX* ctx, X509 *cert);
extern long X_SSL_CTX_set_tmp_ecdh(SSL_CTX* ctx, EC_KEY *key);
extern long X_SSL_CTX_set_tlsext_servername_callback(SSL_CTX* ctx, int (*cb)(SSL *con, int *ad, void *args));
extern int X_SSL_CTX_verify_cb(int ok, X509_STORE_CTX* store);
extern long X_SSL_CTX_set_tmp_dh(SSL_CTX* ctx, DH *dh);
extern long X_PEM_read_DHparams(SSL_CTX* ctx, DH *dh);
extern int X_SSL_CTX_set_tlsext_ticket_key_cb(SSL_CTX *sslctx,
        int (*cb)(SSL *s, unsigned char key_name[16],
                  unsigned char iv[EVP_MAX_IV_LENGTH],
                  EVP_CIPHER_CTX *ctx, HMAC_CTX *hctx, int enc));
extern int X_SSL_CTX_ticket_key_cb(SSL *s, unsigned char key_name[16],
        unsigned char iv[EVP_MAX_IV_LENGTH],
        EVP_CIPHER_CTX *cctx, HMAC_CTX *hctx, int enc);

/* BIO methods */
extern int X_BIO_get_flags(BIO *b);
extern void X_BIO_set_flags(BIO *bio, int flags);
extern void X_BIO_clear_flags(BIO *bio, int flags);
extern void X_BIO_set_data(BIO *bio, void* data);
extern void *X_BIO_get_data(BIO *bio);
extern int X_BIO_read(BIO *b, void *buf, int len);
extern int X_BIO_write(BIO *b, const void *buf, int len);
extern BIO *X_BIO_new_write_bio();
extern BIO *X_BIO_new_read_bio();

/* X509 methods */
extern int X_X509_add_ref(X509* x509);
extern const ASN1_TIME *X_X509_get0_notBefore(const X509 *x);
extern const ASN1_TIME *X_X509_get0_notAfter(const X509 *x);
extern int X_sk_X509_num(STACK_OF(X509) *sk);
extern X509 *X_sk_X509_value(STACK_OF(X509)* sk, int i);
extern long X_X509_get_version(const X509 *x);
extern int X_X509_set_version(X509 *x, long version);

/* PEM methods */
extern int X_PEM_write_bio_PrivateKey_traditional(BIO *bio, EVP_PKEY *key, const EVP_CIPHER *enc, unsigned char *kstr, int klen, pem_password_cb *cb, void *u);

/* cert verify call back */
extern void ssl_ctx_set_hostname_cert_verify_cb(SSL_CTX *ctx, void *arg);

/* cert verify cb about VerifyPeerCertificate*/
extern void ssl_ctx_set_cert_verify_callback_serverVerifyBackForVerifyPeerCertificate(SSL_CTX *ctx, void *arg);
extern void ssl_ctx_set_cert_verify_callback_clientCertVerifyCallBack(SSL_CTX *ctx, void *arg);
extern void ssl_ctx_set_cert_verify_callback_serverCertVerifyCallBack(SSL_CTX *ctx, void *arg);
extern int ssl_cert_verify_call_back_always_success(int i, X509_STORE_CTX *x);
extern void ssl_set_cert_verify_require_peer_cert(SSL *ssl);
extern void ssl_ctx_set_cert_verify_require_peer_cert(SSL_CTX *ctx);

/* clienthello cb about GetConfigForClient*/
extern void ssl_ctx_set_client_hello_cb_GetConfigForClient(SSL_CTX *ctx, void *arg);

/* sni cb about defaultClientServerNameCallBack */
extern void ssl_ctx_set_defaultClientServerNameCallBack(SSL_CTX *ctx);

/* packet extract func */
uint16_t packet_peek_2byte_len(const unsigned char* str);