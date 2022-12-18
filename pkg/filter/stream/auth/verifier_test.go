package auth

import (
	"bytes"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/jsonpb"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	pb "mosn.io/mosn/pkg/filter/stream/auth/matchpb"
	"mosn.io/mosn/pkg/protocol/http"
)

func TestVerifier(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	acceptAuth := NewMockAuthenticator(ctrl)
	acceptAuth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(true).AnyTimes()

	refuseAuth := NewMockAuthenticator(ctrl)
	refuseAuth.EXPECT().Authenticate(gomock.Any(), gomock.Any()).Return(false).AnyTimes()

	authenticators := map[string]Authenticator{
		"accept": acceptAuth,
		"refuse": refuseAuth,
	}

	headers := &http.RequestHeader{
		RequestHeader: &fasthttp.RequestHeader{},
	}

	params := ""

	// AnyVerifier test
	testCases := []struct {
		name        string
		requirement string
		expected    bool
	}{
		{
			name:        "anyVerifier all refuse",
			requirement: anyAllRefuse,
			expected:    false,
		},
		{
			name:        "anyVerifier one accept",
			requirement: anyOneAccept,
			expected:    true,
		},
		{
			name:        "anyVerifier one refuse",
			requirement: anyOneRefuse,
			expected:    true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require := pb.Requirement{}
			if err := jsonpb.Unmarshal(bytes.NewReader([]byte(testCase.requirement)), &require); err != nil {
				t.Errorf("requirement parse error %v", err)
			}
			verifier, err := NewVerifier(&require, authenticators)
			if err != nil {
				t.Errorf("NewVerifier error %v", err)
			}
			ok := verifier.Verify(headers, params)
			assert.Equal(t, testCase.expected, ok)
		})
	}

	// AllVerifier test
	testCases = []struct {
		name        string
		requirement string
		expected    bool
	}{
		{
			name:        "allVerifier all accept",
			requirement: allAllAccept,
			expected:    true,
		},
		{
			name:        "allVerifier one accept",
			requirement: allOneAccept,
			expected:    false,
		},
		{
			name:        "allVerifier one refuse",
			requirement: allOneRefuse,
			expected:    false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			require := pb.Requirement{}
			if err := jsonpb.Unmarshal(bytes.NewReader([]byte(testCase.requirement)), &require); err != nil {
				t.Errorf("requirement parse error %v", err)
			}
			verifier, err := NewVerifier(&require, authenticators)
			if err != nil {
				t.Errorf("NewVerifier error %v", err)
			}
			ok := verifier.Verify(headers, params)
			assert.Equal(t, testCase.expected, ok)
		})
	}
}

var anyAllRefuse = `
	{
		"requires_any": {
			"requirements": [
				{
					"authenticator_name": "refuse"
				},
				{
					"authenticator_name": "refuse"
				},
				{
					"authenticator_name": "refuse"
				}
			]
		}
	}
`

var anyOneAccept = `
	{
		"requires_any": {
			"requirements": [
				{
					"authenticator_name": "accept"
				},
				{
					"authenticator_name": "refuse"
				},
				{
					"authenticator_name": "refuse"
				}
			]
		}
	}
`

var anyOneRefuse = `
	{
		"requires_any": {
			"requirements": [
				{
					"authenticator_name": "refuse"
				},
				{
					"authenticator_name": "accept"
				},
				{
					"authenticator_name": "accept"
				}
			]
		}
	}
`

var allAllAccept = `
	{
		"requires_all": {
			"requirements": [
				{
					"authenticator_name": "accept"
				},
				{
					"authenticator_name": "accept"
				},
				{
					"authenticator_name": "accept"
				}
			]
		}
	}
`

var allOneAccept = `
	{
		"requires_all": {
			"requirements": [
				{
					"authenticator_name": "accept"
				},
				{
					"authenticator_name": "refuse"
				},
				{
					"authenticator_name": "refuse"
				}
			]
		}
	}
`

var allOneRefuse = `
	{
		"requires_all": {
			"requirements": [
				{
					"authenticator_name": "refuse"
				},
				{
					"authenticator_name": "accept"
				},
				{
					"authenticator_name": "accept"
				}
			]
		}
	}
`
