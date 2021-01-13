package jwtauthn

import (
	"testing"
	"time"

	envoycorev3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/lestrrat/go-jwx/jwk"
	"github.com/stretchr/testify/assert"
)

func TestJwksCache(t *testing.T) {
	config, err := getExampleConfig()
	if err != nil {
		t.Errorf("get example config: %v", err)
		t.FailNow()
	}

	cache := NewJwksCache(config.Providers)

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

		cache := NewJwksCache(config.Providers)
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
			LocalJwks: &envoycorev3.DataSource{
				Specifier: &envoycorev3.DataSource_InlineString{
					InlineString: publicKey,
				},
			},
		}

		cache := NewJwksCache(config.Providers)
		jwks := cache.FindByIssuer("https://example.com")
		assert.NotNil(t, jwks.GetJwksObj())
		assert.False(t, jwks.IsExpired())
	})

	t.Run("a bad local jwks.", func(t *testing.T) {
		provider0 := config.Providers[providerName]
		provider0.JwksSourceSpecifier = &jwtauthnv3.JwtProvider_LocalJwks{
			LocalJwks: &envoycorev3.DataSource{
				Specifier: &envoycorev3.DataSource_InlineString{
					InlineString: "BAD-JWKS",
				},
			},
		}

		cache := NewJwksCache(config.Providers)
		jwks := cache.FindByIssuer("https://example.com")
		assert.Nil(t, jwks.GetJwksObj())
	})
}
