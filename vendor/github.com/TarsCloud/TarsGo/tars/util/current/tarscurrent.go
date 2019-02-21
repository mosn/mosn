package current

import "context"

type tarsCurrentKey int64

var tcKey = tarsCurrentKey(0x484900)

//Current contains message for the specify request. This current is used for server side.
type Current struct {
	clientIP   string
	clientPort string
	reqStatus  map[string]string
	resStatus  map[string]string
	reqContext map[string]string
	resContext map[string]string
}

//NewCurrent return a Current point.
func newCurrent() *Current {
	return &Current{}
}

//ContextWithTarsCurrent set TarsCurrent
func ContextWithTarsCurrent(ctx context.Context) context.Context {
	tc := newCurrent()
	ctx = context.WithValue(ctx, tcKey, tc)
	return ctx
}

//GetClientIPFromContext gets the client ip from the context.
func GetClientIPFromContext(ctx context.Context) (string, bool) {
	tc, ok := currentFromContext(ctx)
	if ok {
		return tc.clientIP, ok
	}
	return "", ok
}

//SetClientIPWithContext set Client IP to the tars current.
func SetClientIPWithContext(ctx context.Context, IP string) bool {
	tc, ok := currentFromContext(ctx)
	if ok {
		tc.clientIP = IP
	}
	return ok
}

//GetClientPortFromContext gets the client ip from the context.
func GetClientPortFromContext(ctx context.Context) (string, bool) {
	tc, ok := currentFromContext(ctx)
	if ok {
		return tc.clientPort, ok
	}
	return "", ok
}

//SetClientPortWithContext set client port to the tars current.
func SetClientPortWithContext(ctx context.Context, port string) bool {
	tc, ok := currentFromContext(ctx)
	if ok {
		tc.clientPort = port
	}
	return ok
}

//currentFromContext gets current from the context
func currentFromContext(ctx context.Context) (*Current, bool) {
	tc, ok := ctx.Value(tcKey).(*Current)
	return tc, ok
}

//SetResponseStatus set the response package' status .
func SetResponseStatus(ctx context.Context, s map[string]string) bool {
	tc, ok := currentFromContext(ctx)
	if ok {
		tc.resStatus = s
	}
	return ok
}

//GetResponseStatus get response status set by user.
func GetResponseStatus(ctx context.Context) (map[string]string, bool) {
	tc, ok := currentFromContext(ctx)
	if ok {
		return tc.resStatus, ok
	}
	return nil, ok
}

//SetResponseContext set the response package' context .
func SetResponseContext(ctx context.Context, c map[string]string) bool {
	tc, ok := currentFromContext(ctx)
	if ok {
		tc.resContext = c
	}
	return ok
}

//GetResponseContext get response context set by user.
func GetResponseContext(ctx context.Context) (map[string]string, bool) {
	tc, ok := currentFromContext(ctx)
	if ok {
		return tc.resContext, ok
	}
	return nil, ok
}

//SetRequestStatus set the request package' status .
func SetRequestStatus(ctx context.Context, s map[string]string) bool {
	tc, ok := currentFromContext(ctx)
	if ok {
		tc.reqStatus = s
	}
	return ok
}

//GetRequestStatus get request status set by user.
func GetRequestStatus(ctx context.Context) (map[string]string, bool) {
	tc, ok := currentFromContext(ctx)
	if ok {
		return tc.reqStatus, ok
	}
	return nil, ok
}

//SetRequestContext set the request package' context .
func SetRequestContext(ctx context.Context, c map[string]string) bool {
	tc, ok := currentFromContext(ctx)
	if ok {
		tc.reqContext = c
	}
	return ok
}

//GetRequestContext get request context set by user.
func GetRequestContext(ctx context.Context) (map[string]string, bool) {
	tc, ok := currentFromContext(ctx)
	if ok {
		return tc.reqContext, ok
	}
	return nil, ok
}
