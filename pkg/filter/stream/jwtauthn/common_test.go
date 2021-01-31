package jwtauthn

import (
	jwtauthnv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/valyala/fasthttp"
	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol/http"
)

// publicKey is a good public key.
var publicKey = `
{
  "keys": [
    {
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "kid": "62a93512c9ee4c7f8067b5a216dade2763d32a47",
      "n": "up97uqrF9MWOPaPkwSaBeuAPLOr9FKcaWGdVEGzQ4f3Zq5WKVZowx9TCBxmImNJ1qmUi13pB8otwM_l5lfY1AFBMxVbQCUXntLovhDaiSvYp4wGDjFzQiYA-pUq8h6MUZBnhleYrkU7XlCBwNVyN8qNMkpLA7KFZYz-486GnV2NIJJx_4BGa3HdKwQGxi2tjuQsQvao5W4xmSVaaEWopBwMy2QmlhSFQuPUpTaywTqUcUq_6SfAHhZ4IDa_FxEd2c2z8gFGtfst9cY3lRYf-c_ZdboY3mqN9Su3-j3z5r2SHWlhB_LNAjyWlBGsvbGPlTqDziYQwZN4aGsqVKQb9Vw",
      "e": "AQAB"
    },
    {
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "kid": "b3319a147514df7ee5e4bcdee51350cc890cc89e",
      "n": "up97uqrF9MWOPaPkwSaBeuAPLOr9FKcaWGdVEGzQ4f3Zq5WKVZowx9TCBxmImNJ1qmUi13pB8otwM_l5lfY1AFBMxVbQCUXntLovhDaiSvYp4wGDjFzQiYA-pUq8h6MUZBnhleYrkU7XlCBwNVyN8qNMkpLA7KFZYz-486GnV2NIJJx_4BGa3HdKwQGxi2tjuQsQvao5W4xmSVaaEWopBwMy2QmlhSFQuPUpTaywTqUcUq_6SfAHhZ4IDa_FxEd2c2z8gFGtfst9cY3lRYf-c_ZdboY3mqN9Su3-j3z5r2SHWlhB_LNAjyWlBGsvbGPlTqDziYQwZN4aGsqVKQb9Vw",
      "e": "AQAB"
    }
  ]
}
`

var providerName = "example_provider"

// A good config.
var exampleConfig = `
{
  "providers": {
    "example_provider": {
      "issuer": "https://example.com",
      "audiences": [
        "example_service",
        "http://example_service1",
        "https://example_service2/"
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
  },
  "rules": [
    {
      "match": {
        "path": "/"
      },
      "requires": {
        "provider_name": "example_provider"
      }
    }
  ],
  "bypass_cors_preflight": true
}
`

// {"iss":"https://example.com","sub":"test@example.com","aud":"example_service","exp":2001001001}
var goodToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjYyYTkzNTEyYzllZTRjN2Y4MDY3YjVhMjE2ZGFkZTI3NjNkMzJhNDciLCJ0eXAiOiJKV1QifQ.eyJhdWQiOiJleGFtcGxlX3NlcnZpY2UiLCJleHAiOjIwMDEwMDEwMDEsImlzcyI6Imh0dHBzOi8vZXhhbXBsZS5jb20iLCJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIn0.ExL4uq6uJuAlvOtg8ixccKRvZIjekZIC5e_0OotJjAVT3NFckA5qtyfXiuizQ3E2Af04bkWBkwP6vj6jmrM_teRsMrAvNojWLOHQ4al2YbIZ0kjAv0Y9FGqsf6H5_eOFTcC1Vjh-Ou22Wf_7tyzgRbMGgmTS9BkFR4Sgmsa5mvQpjoUAOZD-M9TmO8eWYCX7BSqBdftafDaYOkJDeyAlWnzzmb3-1UZz7iokS7Ra_Cu74HxBlP_1TiA0_vp0sYcA_sZkDQEWVaqhub63gMKnFvMBAr3--Z4ImqLZvdP2DO59jP_mVV9tazgBKEBVFDL9RYUwrLfQVjCM_qbCMXLr8g"

// Expected base64 payload value.
var expectedPayloadValue = "eyJhdWQiOiJleGFtcGxlX3NlcnZpY2UiLCJleHAiOjIwMDEwMDEwMDEsImlzcyI6Imh0dHBzOi8vZXhhbXBsZS5jb20iLCJzdWIiOiJ0ZXN0QGV4YW1wbGUuY29tIn0"

// {"iss":"https://example.com","sub":"test@example.com","aud":"example_service","exp":2001001001}
var nonExistKIDToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL2V4YW1wbGUuY29tIiwic3ViIjoidGVzdEBleGFtcGxlLmNvbSIsImF1ZCI6ImV4YW1wbGVfc2VydmljZSIsImV4cCI6MjAwMTAwMTAwMX0.n45uWZfIBZwCIPiL0K8Ca3tmm-ZlsDrC79_vXCspPwk5oxdSn983tuC9GfVWKXWUMHe11DsB02b19Ow-fmoEzooTFn65Ml7G34nW07amyM6lETiMhNzyiunctplOr6xKKJHmzTUhfTirvDeG-q9n24-8lH7GP8GgHvDlgSM9OY7TGp81bRcnZBmxim_UzHoYO3_c8OP4ZX3xG5PfihVk5G0g6wcHrO70w0_64JgkKRCrLHMJSrhIgp9NHel_CNOnL0AjQKe9IGblJrMuouqYYS0zEWwmOVUWUSxQkoLpldQUVefcfjQeGjz8IlvktRa77FYexfP590ACPyXrivtsxg"

// {"iss":"https://example.com","sub":"test@example.com","exp":null}
var nonExpiringToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL2V4YW1wbGUuY29tIiwic3ViIjoidGVzdEBleGFtcGxlLmNvbSIsImlhdCI6MTUzMzE3NTk0Mn0.OSh-AcY9dCUibXiIZzPlTdEsYH8xP3QkCJDesO3LVu4ndgTrxDnNuR3I4_oV4tjtirmLZD3sx96wmLiIhOyqj3nipIdf_aQWcmET0XoRqGixOKse5FlHyU_VC1Jj9AlMvSz9zyCvKxMyP0CeA-bhI_Qs-I9vBPK8pd-EUOespUqWMQwNdtrOdXLcvF8EA5BV5G2qRGzCU0QJaW0DpyjYF7ZCswRGorc2oMt5duXSp3-L1b9dDrnLwroxUrmQIZz9qvfwdDR-guyYSjKVQu5NJAyysd8XKNzmHqJ2fYhRjc5s7l5nIWTDyBXSdPKQ8cBnfFKoxaRhmMBjdEn9RB7r6A"

// {"iss":"https://example.com","sub":"test@example.com","aud":"example_service","exp":1205005587}
var expiredToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL2V4YW1wbGUuY29tIiwic3ViIjoidGVzdEBleGFtcGxlLmNvbSIsImV4cCI6MTIwNTAwNTU4NywiYXVkIjoiZXhhbXBsZV9zZXJ2aWNlIn0.izDa6aHNgbsbeRzucE0baXIP7SXOrgopYQALLFAsKq_N0GvOyqpAZA9nwCAhqCkeKWcL-9gbQe3XJa0KN3FPa2NbW4ChenIjmf2QYXOuOQaDu9QRTdHEY2Y4mRy6DiTZAsBHWGA71_cLX-rzTSO_8aC8eIqdHo898oJw3E8ISKdryYjayb9X3wtF6KLgNomoD9_nqtOkliuLElD8grO0qHKI1xQurGZNaoeyiV1AdwgX_5n3SmQTacVN0WcSgk6YJRZG6VE8PjxZP9bEameBmbSB0810giKRpdTU1-RJtjq6aCSTD4CYXtW38T5uko4V-S4zifK3BXeituUTebkgoA"

// {"iss":"https://example.com","sub":"test@example.com","aud":"example_service","nbf":9223372036854775807}
var notYetValidToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL2V4YW1wbGUuY29tIiwic3ViIjoidGVzdEBleGFtcGxlLmNvbSIsImF1ZCI6ImV4YW1wbGVfc2VydmljZSIsIm5iZiI6OTIyMzM3MjAzNjg1NDc3NTgwN30K.izDa6aHNgbsbeRzucE0baXIP7SXOrgopYQALLFAsKq_N0GvOyqpAZA9nwCAhqCkeKWcL-9gbQe3XJa0KN3FPa2NbW4ChenIjmf2QYXOuOQaDu9QRTdHEY2Y4mRy6DiTZAsBHWGA71_cLX-rzTSO_8aC8eIqdHo898oJw3E8ISKdryYjayb9X3wtF6KLgNomoD9_nqtOkliuLElD8grO0qHKI1xQurGZNaoeyiV1AdwgX_5n3SmQTacVN0WcSgk6YJRZG6VE8PjxZP9bEameBmbSB0810giKRpdTU1-RJtjq6aCSTD4CYXtW38T5uko4V-S4zifK3BXeituUTebkgoA"

// {"iss":"https://example.com","sub":"test@example.com","aud":"invalid_service","exp":2001001001}
var invalidAudToken = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJodHRwczovL2V4YW1wbGUuY29tIiwic3ViIjoidGVzdEBleGFtcGxlLmNvbSIsImV4cCI6MjAwMTAwMTAwMSwiYXVkIjoiaW52YWxpZF9zZXJ2aWNlIn0.B9HuVXpRDVYIvApfNQmE_l5fEMPEiPdi-sdKbTione8I_UsnYHccKZVegaF6f2uyWhAvaTPgaMosyDlJD6skadEcmZD0V4TzsYKv7eP5FQga26hZ1Kra7n9hAq4oFfH0J8aZLOvDV3tAgCNRXlh9h7QiBPeDNQlwztqEcsyp1lHI3jdUhsn3InIn-vathdx4PWQWLVb-74vwsP-END-MGlOfu_TY5OZUeY-GBE4Wr06aOSU2XQjuNr6y2WJGMYFsKKWfF01kHSuyc9hjnq5UI19WrOM8s7LFP4w2iKWFIPUGmPy3aM0TiF2oFOuuMxdPR3HNdSG7EWWRwoXv7n__jA"

// {"iss":"https://other.com","sub":"test@other.com","aud":"other_service","exp":2001001001}
var otherGoodToken = "eyJhbGciOiJSUzI1NiIsImtpZCI6IjYyYTkzNTEyYzllZTRjN2Y4MDY3YjVhMjE2ZGFkZTI3NjNkMzJhNDciLCJ0eXAiOiJKV1QifQ.eyJhdWQiOiJvdGhlcl9zZXJ2aWNlIiwiZXhwIjoyMDAxMDAxMDAxLCJpc3MiOiJodHRwczovL290aGVyLmNvbSIsInN1YiI6InRlc3RAb3RoZXIuY29tIn0.YGgYFjYObwjLqjqRkEU1u1MY3s959ttgpXIFKHWwe-fp1phnQj7kyS8p6scmibnpuy7PWnLklRzu4TbLVbUpfmyrkDprjBrHB6mriLpXt8-7trwPAWKmpG2fKybB_iz7Uj9Qsy4XjbNBpw_c3VBapifc2e9qcPb2U6SfJVYL0ixyTpIhdQnZufvnk3cBpsF3TCQK0WQULS4-5vucYry1SmnwJLKdYX4sLF23dTFS_IFNHPjP46Mk0s-8qud9KCVssk_fUu7fVksTOkXOXSNzAcqlNOAbAfGaiT-32wZHMTmJzasvuyRZ1uHKLnxgb-mF9eUcjsFpeLpSLtDivaq9uQ"

func newHeaders(headers ...[2]string) api.HeaderMap {
	res := &http.RequestHeader{
		RequestHeader: &fasthttp.RequestHeader{},
	}
	for _, header := range headers {
		res.Set(header[0], header[1])
	}
	return res
}

func addHeader(k, v string) api.HeaderMap {
	header := &http.RequestHeader{
		RequestHeader: &fasthttp.RequestHeader{},
	}
	header.Add(k, v)
	return header
}

func getExampleConfig() (*jwtauthnv3.JwtAuthentication, error) {
	var config jwtauthnv3.JwtAuthentication
	if err := jsonpb.UnmarshalString(exampleConfig, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
