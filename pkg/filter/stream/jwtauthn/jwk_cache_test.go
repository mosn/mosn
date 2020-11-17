package jwtauthn

import (
	"crypto/rand"
	"crypto/rsa"
	"github.com/lestrrat/go-jwx/jwa"
	"github.com/lestrrat/go-jwx/jwt"
	"testing"
	"time"

	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/lestrrat/go-jwx/jwk"
	"github.com/stretchr/testify/assert"
)

func TestJwksCache(t *testing.T) {
	var config jwtauthnv3.JwtAuthentication
	if err := jsonpb.UnmarshalString(exampleConfig, &config); err != nil {
		t.Errorf("unmarshal exampleConfig to config(jwtauthnv3.JwtAuthentication): %v", err)
		t.FailNow()
	}
	provider := config.Providers[providerName]
	remoteJwks := &jwtauthnv3.RemoteJwks{}
	remoteJwks.HttpUri = &envoy_config_core_v3.HttpUri{
		Uri: "https://pubkey_server/pubkey_path",
		Timeout: &duration.Duration{
			Seconds: 5,
		},
		HttpUpstreamType: &envoy_config_core_v3.HttpUri_Cluster{
			Cluster: "pubkey_cluster",
		},
	}
	remoteJwks.CacheDuration = &duration.Duration{
		Seconds: 600,
	}
	provider.JwksSourceSpecifier = &jwtauthnv3.JwtProvider_RemoteJwks{
		RemoteJwks: remoteJwks,
	}

	cache := NewJwksCacheDeprecated(&config)

	t.Run("findByIssuer", func(t *testing.T) {
		assert.NotNil(t, cache.FindByIssuer("https://example.com"))
		assert.Nil(t, cache.FindByIssuer("https://foobar.com"))
	})

	t.Run("findByProvider", func(t *testing.T) {
		assert.NotNil(t, cache.FindByProvider(providerName))
		assert.Nil(t, cache.FindByProvider("other-provider"))
	})

	t.Run("setRemoteJwks and its expiration", func(t *testing.T) {
		provider0 := config.Providers[providerName]
		// Set cache_duration to 1 second to test expiration
		provider0.GetRemoteJwks().CacheDuration = &duration.Duration{
			Seconds: 1,
		}

		cache := NewJwksCacheDeprecated(&config)
		jwks := cache.FindByIssuer("https://example.com")
		assert.Nil(t, jwks.GetJwksObj())

		set, _ := jwk.ParseString(publicKey)
		jwks.SetRemoteJwks(set)
		assert.NotNil(t, jwks.GetJwksObj())
		assert.False(t, jwks.IsExpired())

		// cache duration is 1 second, sleep two seconds to expire it
		time.Sleep(2 * time.Second)
		assert.True(t, jwks.IsExpired())
	})

	t.Run("a good local jwks.", func(t *testing.T) {
		provider0 := config.Providers[providerName]
		provider0.JwksSourceSpecifier = &jwtauthnv3.JwtProvider_LocalJwks{
			LocalJwks: &envoy_config_core_v3.DataSource{
				Specifier: &envoy_config_core_v3.DataSource_InlineString{
					InlineString: publicKey,
				},
			},
		}

		cache := NewJwksCacheDeprecated(&config)
		jwks := cache.FindByIssuer("https://example.com")
		assert.NotNil(t, jwks.GetJwksObj())
		assert.False(t, jwks.IsExpired())
	})

	t.Run("a bad local jwks.", func(t *testing.T) {
		provider0 := config.Providers[providerName]
		provider0.JwksSourceSpecifier = &jwtauthnv3.JwtProvider_LocalJwks{
			LocalJwks: &envoy_config_core_v3.DataSource{
				Specifier: &envoy_config_core_v3.DataSource_InlineString{
					InlineString: "BAD-JWKS",
				},
			},
		}

		cache := NewJwksCacheDeprecated(&config)
		jwks := cache.FindByIssuer("https://example.com")
		assert.Nil(t, jwks.GetJwksObj())
	})
}

func TestFoobar2(t *testing.T) {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return
	}

	token := jwt.New()
	token.Set(`foo`, `bar`)
	bytes, err := token.Sign(jwa.RS256, privKey)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(string(bytes))
	t.Log(privKey.PublicKey)
}
