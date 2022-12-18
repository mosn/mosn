package keyauth

import (
	"github.com/valyala/fasthttp"
	"mosn.io/api"
	pb "mosn.io/mosn/pkg/filter/stream/auth/keyauth/keyauthpb"
)

type empty struct{}

type KeyAuth struct {
	FromHeader string
	FromParam  string
	Keys       map[string]empty
}

func NewKeyAuth(config *pb.Authenticator) KeyAuth {
	keys := make(map[string]empty, len(config.Keys))
	for _, key := range config.Keys {
		keys[key] = empty{}
	}

	return KeyAuth{
		FromHeader: config.FromHeader,
		FromParam:  config.FromParam,
		Keys:       keys,
	}
}

func (a KeyAuth) Authenticate(headers api.HeaderMap, requestArg string) bool {

	// Get the key from header
	if a.FromHeader != "" {
		key, ok := headers.Get(a.FromHeader)
		if ok {
			_, exist := a.Keys[key]
			if exist {
				return true
			}
		}
	}

	// Get the key from requestArg
	if a.FromParam != "" {
		args := fasthttp.AcquireArgs()
		args.Parse(requestArg)
		key := string(args.Peek(a.FromParam))

		_, exist := a.Keys[key]
		if exist {
			return true
		}
	}

	return false
}
