package jwtauthn

import (
	"testing"

	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/jsonpb"
	"github.com/lestrrat/go-jwx/jwk"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticatorVerify(t *testing.T) {
	getConfig := func() *jwtauthnv3.JwtAuthentication {
		config, err := getExampleConfig()
		if err != nil {
			t.Errorf("get example config: %v", err)
			t.FailNow()
		}
		return config
	}

	getExtractor := func(config *jwtauthnv3.JwtAuthentication) Extractor {
		var providers []*jwtauthnv3.JwtProvider
		for _, pro := range config.Providers {
			providers = append(providers, pro)
		}
		return NewExtractor(providers)
	}

	jwks, _ := jwk.Parse([]byte(publicKey))

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	t.Run("A good JWT authentication with a remote Jwks.", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+goodToken)
		config := getConfig()
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		// call Fetch once
		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		for i := 0; i < 10; i++ {
			err := auth.Verify(headers, tokens)
			if err != nil {
				t.Error(err)
				return
			}

			gotPayload, _ := headers.Get("sec-istio-auth-userinfo")
			assert.Equal(t, expectedPayloadValue, gotPayload)

			// Verify the token is removed.
			_, exists := headers.Get("Authorization")
			assert.False(t, exists)
		}
	})

	t.Run("Jwt is forwarded if 'forward' flag is set.", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+goodToken)
		config := getConfig()
		provider := config.Providers[providerName]
		provider.Forward = true
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		if err != nil {
			t.Error(err)
		}

		// Verify the token is NOT removed.
		_, exists := headers.Get("Authorization")
		assert.True(t, exists)
	})

	t.Run("Jwt with non existing kid", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+nonExistKIDToken)
		config := getConfig()
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrInvalidToken, err)
	})

	t.Run("Jwt is missing, proper status is called", func(t *testing.T) {
		headers := newHeaders()
		config := getConfig()
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(0)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrJwtNotFound, err)
	})

	t.Run("Jwt is invalid, ErrJwtBadFormat is returned", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer invalidToken")
		config := getConfig()
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(0)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrJwtBadFormat, err)
	})

	t.Run("Authorization header has invalid prefix, ErrJwtNotFound is returned", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer-invalid")
		config := getConfig()
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(0)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrJwtNotFound, err)
	})

	t.Run("When a JWT is non-expiring without audience specified, ErrJwtAudienceNotAllowed is returned.", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+nonExpiringToken)
		config := getConfig()
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(0)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrJwtAudienceNotAllowed, err)
	})

	t.Run("A JWT is expired, ErrJwtExpired status is returned.", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+expiredToken)
		config := getConfig()
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(0)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrJwtExpired, err)
	})

	t.Run("When a JWT is not yet valid, ErrJwtNotYetValid status is returned.", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+notYetValidToken)
		config := getConfig()
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(0)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrJwtNotYetValid, err)
	})

	t.Run("When an inline JWKS is misconfigured, JwksNoValidKeys is returns", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+goodToken)
		config := getConfig()
		provider := config.Providers[providerName]
		provider.JwksSourceSpecifier = &jwtauthnv3.JwtProvider_LocalJwks{
			LocalJwks: &envoycorev3.DataSource{
				Specifier: &envoycorev3.DataSource_InlineString{
					InlineString: "invalid",
				},
			},
		}
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(0)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrJwksNoValidKeys, err)
	})

	t.Run("When a JWT is with invalid audience, ErrJwtAudienceNotAllowed is returned", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+invalidAudToken)
		config := getConfig()
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(0)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrJwtAudienceNotAllowed, err)
	})

	t.Run("When Jwt issuer is not configured, ErrJwtUnknownIssuer is returned.", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+goodToken)
		config := getConfig()
		provider := config.Providers[providerName]
		provider.Issuer = "other_issuer"
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(0)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrJwtUnknownIssuer, err)
	})

	t.Run("When Jwks fetching fails, ErrJwksFetch status is returned.", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+goodToken)
		config := getConfig()
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(nil, ErrJwksFetch).Times(1)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Equal(t, ErrJwksFetch, err)
	})

	t.Run("If 'forward_payload_header' is empty, payload is not forwarded.", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer "+goodToken)
		config := getConfig()
		provider0 := config.Providers[providerName]
		provider0.ForwardPayloadHeader = ""
		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Nil(t, err)
		_, exists := headers.Get("sec-istio-auth-userinfo")
		assert.False(t, exists)
	})

	t.Run("Allow failed authenticator will verify all tokens.", func(t *testing.T) {
		config := getConfig()
		provider0 := config.Providers[providerName]
		names := []string{"a", "b", "c"}
		for _, name := range names {
			provider0.FromHeaders = append(provider0.FromHeaders, &jwtauthnv3.JwtHeader{
				Name:        name,
				ValuePrefix: "Bearer ",
			})
		}

		headers := newHeaders()
		headers.Add("a", "Bearer "+expiredToken)
		headers.Add("b", "Bearer "+goodToken)
		headers.Add("c", "Bearer "+invalidAudToken)

		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator(providerName, jwksCache, jwksFetcher, false, false)
		err := auth.Verify(headers, tokens)
		assert.Nil(t, err)
		_, exists := headers.Get("a")
		assert.True(t, exists)
		_, exists = headers.Get("b")
		assert.False(t, exists)
		_, exists = headers.Get("c")
		assert.True(t, exists)

		headers = newHeaders()
		headers.Add("a", "Bearer "+goodToken)
		headers.Add("b", "Bearer "+goodToken)
		headers.Add("c", "Bearer "+goodToken)

		extractor = getExtractor(config)
		tokens = extractor.Extract(headers, "")

		jwksFetcher = NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(0)

		auth = newAuthenticator(providerName, jwksCache, jwksFetcher, true, false)
		err = auth.Verify(headers, tokens)
		assert.Nil(t, err)
		_, exists = headers.Get("a")
		assert.False(t, exists)
		_, exists = headers.Get("b")
		assert.False(t, exists)
		_, exists = headers.Get("c")
		assert.False(t, exists)
	})

	t.Run("Allow failed authenticator will verify all tokens.", func(t *testing.T) {
		config := getConfig()
		otherProviderConfig := `
{
  "issuer": "https://other.com",
  "audiences": [
	"other_service"
  ],
  "from_headers": [
	{
	   "name": "expired-auth",
	   "value_prefix": "Bearer "
	},
    {
	   "name": "other-auth",
	   "value_prefix": "Bearer "
	}
  ],
  "remote_jwks": {
	"http_uri": {
	  "uri": "https://pubkey_server/pubkey_path",
	  "cluster": "pubkey_cluster",
	  "timeout": "5s"
	},
	"cache_duration": "600s"
  },
  "forward_payload_header": "sec-istio-auth-userinfo"
}
`
		var provider jwtauthnv3.JwtProvider
		err := jsonpb.UnmarshalString(otherProviderConfig, &provider)
		assert.Nil(t, err)
		config.Providers["other_provider"] = &provider

		headers := newHeaders()
		headers.Add("Authorization", "Bearer "+goodToken)
		headers.Add("expired-auth", "Bearer "+expiredToken)
		headers.Add("other-auth", "Bearer "+otherGoodToken)

		extractor := getExtractor(config)
		tokens := extractor.Extract(headers, "")

		jwksFetcher := NewMockJwksFetcher(ctrl)
		jwksFetcher.EXPECT().Fetch(gomock.Any()).Return(jwks, nil).Times(2)

		jwksCache := NewJwksCache(config.Providers)
		auth := newAuthenticator("", jwksCache, jwksFetcher, true, true)
		err = auth.Verify(headers, tokens)
		assert.Nil(t, err)
		_, exists := headers.Get("Authorization")
		assert.False(t, exists)
		_, exists = headers.Get("expired-auth")
		assert.True(t, exists)
		_, exists = headers.Get("other-auth")
		assert.False(t, exists)
	})
}
