package auth

import (
	"mosn.io/api"
)

// Authenticator object to handle all authentication flow.
type Authenticator interface {
	Authenticate(headers api.HeaderMap, requestArg string) bool
}
