package jwtauthn

import (
	"encoding/json"
	"sort"
	"testing"

	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"github.com/stretchr/testify/assert"
)

func TestExtractorExtract(t *testing.T) {
	exampleConfig := `
{
   "providers": {
      "provider1": {
         "issuer": "issuer1"
      },
      "provider2": {
         "issuer": "issuer2",
         "from_headers": [
            {
               "name": "token-header"
            }
         ]
      },
      "provider3": {
         "issuer": "issuer3",
         "from_params": [
            "token_param"
         ]
      },
      "provider4": {
         "issuer": "issuer4",
         "from_headers": [
            {
               "name": "token-header"
            }
         ],
         "from_params": [
            "token_param"
         ]
      },
      "provider5": {
         "issuer": "issuer5",
         "from_headers": [
            {
               "name": "prefix-header",
               "value_prefix": "AAA"
            }
         ]
      },
      "provider6": {
         "issuer": "issuer6",
         "from_headers": [
            {
               "name": "prefix-header",
               "value_prefix": "AAABBB"
            }
         ]
      },
      "provider7": {
         "issuer": "issuer7",
         "from_headers": [
            {
               "name": "prefix-header",
               "value_prefix": "CCCDDD"
            }
         ]
      },
      "provider8": {
         "issuer": "issuer8",
         "from_headers": [
            {
               "name": "prefix-header",
               "value_prefix": "\"CCCDDD\""
            }
         ]
      }
   }
}
`

	var config jwtauthnv3.JwtAuthentication
	if err := json.Unmarshal([]byte(exampleConfig), &config); err != nil {
		t.Errorf("unmarshal exampleConfig to config(jwtauthnv3.JwtAuthentication): %v", err)
		t.FailNow()
	}

	var providers []*jwtauthnv3.JwtProvider
	for _, pro := range config.Providers {
		providers = append(providers, pro)
	}

	extractor := NewExtractor(providers)

	t.Run("not token in the request headers", func(t *testing.T) {
		headers := newHeaders()
		tokens := extractor.Extract(headers, "")
		assert.Equal(t, len(tokens), 0)
	})

	t.Run("the token in the wrong header.", func(t *testing.T) {
		headers := addHeader("wrong-token-header", "jwt_token")
		tokens := extractor.Extract(headers, "")
		assert.Equal(t, len(tokens), 0)
	})

	t.Run("the token in the wrong query parameter.", func(t *testing.T) {
		headers := newHeaders()
		tokens := extractor.Extract(headers, "wrong_token=jwt_token")
		assert.Equal(t, len(tokens), 0)
	})

	t.Run("extracting token from the default header location: 'Authorization'", func(t *testing.T) {
		headers := addHeader("Authorization", "Bearer jwt_token")
		tokens := extractor.Extract(headers, "")
		assert.Equal(t, len(tokens), 1)

		// Only the issue1 is using default header location.
		assert.Equal(t, tokens[0].Token(), "jwt_token")
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer1"), true)

		// Other issuers are using custom locations
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer2"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer3"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer4"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer5"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("unknown_issuer"), false)

		// Test token remove
		tokens[0].RemoveJwt(headers)
		_, ok := headers.Get(authorization)
		assert.Equal(t, ok, false)
	})

	t.Run("extracting token from the default query parameter: 'access_token'", func(t *testing.T) {
		headers := newHeaders()
		requestArg := "access_token=jwt_token"
		tokens := extractor.Extract(headers, requestArg)
		assert.Equal(t, len(tokens), 1)

		// Only the issue1 is using default header location.
		assert.Equal(t, tokens[0].Token(), "jwt_token")
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer1"), true)

		// Other issuers are using custom locations
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer2"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer3"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer4"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer5"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("unknown_issuer"), false)
	})

	t.Run("extracting token from the custom header location: 'token-header'", func(t *testing.T) {
		headers := addHeader("token-header", "jwt_token")
		tokens := extractor.Extract(headers, "")
		assert.Equal(t, len(tokens), 1)

		// Only the issue1 is using default header location.
		assert.Equal(t, tokens[0].Token(), "jwt_token")
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer2"), true)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer4"), true)

		// Other issuers are using custom locations
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer1"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer3"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer5"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("unknown_issuer"), false)

		// Test token remove
		tokens[0].RemoveJwt(headers)
		_, ok := headers.Get(authorization)
		assert.Equal(t, ok, false)
	})

	t.Run("extracting token from the custom header: 'prefix-header'. value prefix doesn't match. It has to be either 'AAA' or 'AAABBB'.", func(t *testing.T) {
		headers := addHeader("prefix-header", "jwt_token")
		tokens := extractor.Extract(headers, "")
		assert.Equal(t, len(tokens), 0)
	})

	t.Run("extracting token from the custom header: 'prefix-header'. The value matches both prefix values: 'AAA' or 'AAABBB'.", func(t *testing.T) {
		headers := addHeader("prefix-header", "AAABBBjwt_token")
		tokens := extractor.Extract(headers, "")
		assert.Equal(t, len(tokens), 2)

		// Match issuer 5 with map key as: prefix-header + AAA
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer5"), true)
		assert.Equal(t, tokens[0].Token(), "BBBjwt_token")

		// Match issuer 6 with map key as: prefix-header + AAABBB which is after AAA
		assert.Equal(t, tokens[1].IsIssuerSpecified("issuer6"), true)
		assert.Equal(t, tokens[1].Token(), "jwt_token")

		tokens[0].RemoveJwt(headers)
		_, ok := headers.Get(authorization)
		assert.Equal(t, ok, false)
	})

	t.Run("extracting token from the custom header: 'prefix-header'. The value is found after the 'CCCDDD', then between the '=' and the ','.", func(t *testing.T) {
		headers := addHeader("prefix-header", "AAABBBjwt_token")
		tokens := extractor.Extract(headers, "")
		assert.Equal(t, len(tokens), 2)

		// Match issuer 5 with map key as: prefix-header + AAA
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer5"), true)
		assert.Equal(t, tokens[0].Token(), "BBBjwt_token")

		// Match issuer 6 with map key as: prefix-header + AAABBB which is after AAA
		assert.Equal(t, tokens[1].IsIssuerSpecified("issuer6"), true)
		assert.Equal(t, tokens[1].Token(), "jwt_token")

		tokens[0].RemoveJwt(headers)
		_, ok := headers.Get(authorization)
		assert.Equal(t, ok, false)
	})

	t.Run("extracting token from the custom query parameter: 'token_param'.", func(t *testing.T) {
		headers := newHeaders()
		requestArg := "token_param=jwt_token"
		tokens := extractor.Extract(headers, requestArg)
		assert.Equal(t, len(tokens), 1)

		// Both issuer3 and issuer4 have specified this custom query location.
		assert.Equal(t, tokens[0].Token(), "jwt_token")
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer3"), true)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer4"), true)

		// Other issuers are using custom locations
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer1"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer2"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("issuer5"), false)
		assert.Equal(t, tokens[0].IsIssuerSpecified("unknown_issuer"), false)
	})

	t.Run("extracting multiple tokens.", func(t *testing.T) {
		headers := addHeader("token-header", "token2")
		headers.Add("authorization", "Bearer token1")
		headers.Add("prefix-header", "AAAtoken5")
		requestArg := "token_param=token3&access_token=token4"
		tokens := extractor.Extract(headers, requestArg)
		assert.Equal(t, len(tokens), 5)

		var tokenStrs []string
		for _, token := range tokens {
			tokenStrs = append(tokenStrs, token.Token())
		}
		sort.Strings(tokenStrs)

		assert.Equal(t, tokenStrs[0], "token1")
		assert.Equal(t, tokenStrs[1], "token2")
		assert.Equal(t, tokenStrs[2], "token3")
		assert.Equal(t, tokenStrs[3], "token4")
		assert.Equal(t, tokenStrs[4], "token5")
	})

	t.Run("selected extraction of multiple tokens.", func(t *testing.T) {
		headers := addHeader("token-header", "token2")
		headers.Add("authorization", "Bearer token1")
		headers.Add("prefix-header", "AAAtoken5")
		requestArg := "token_param=token3&access_token=token4"

		provider := &jwtauthnv3.JwtProvider{
			Issuer: "foo",
		}
		extractor := NewExtractor([]*jwtauthnv3.JwtProvider{provider})
		tokens := extractor.Extract(headers, requestArg)
		assert.Equal(t, len(tokens), 2)
		assert.Equal(t, tokens[0].Token(), "token1")
		assert.Equal(t, tokens[1].Token(), "token4")

		provider.FromHeaders = append(provider.FromHeaders, &jwtauthnv3.JwtHeader{
			Name:        "prefix-header",
			ValuePrefix: "AAA",
		})
		provider.FromParams = append(provider.FromParams, "token_param")
		extractor = NewExtractor([]*jwtauthnv3.JwtProvider{provider})
		tokens = extractor.Extract(headers, requestArg)
		assert.Equal(t, len(tokens), 2)
		assert.Equal(t, tokens[0].Token(), "token5")
		assert.Equal(t, tokens[1].Token(), "token3")
	})
}
