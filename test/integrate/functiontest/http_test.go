package functiontest

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/stream/http"
	_ "mosn.io/mosn/pkg/stream/http2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/test/integrate"
	"mosn.io/mosn/test/util"
)

// check request path is match method
type MethodHTTPHandler struct {
	latency func() time.Duration
}

type errorWrapper struct {
	method   string
	original error
}

func (e errorWrapper) Error() string {
	return fmt.Sprintf("method:%s request error: %v", e.method, e.original)
}

func (e errorWrapper) Timeout() bool {
	err, ok := e.original.(net.Error)
	return ok && err.Timeout()
}

func (h *MethodHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.latency != nil {
		if l := h.latency(); l > 0 {
			time.Sleep(l)
		}
	}
	m := r.Method
	mm := strings.Trim(r.URL.Path, "/")
	w.Header().Set("Content-Type", "text/plain")
	if m != mm {
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// http.MethodConnect is quite different from other method
// http.MethodHead have bugs on http2, needs to fix
var allMethods = []string{http.MethodDelete, http.MethodGet, http.MethodOptions, http.MethodPatch, http.MethodPost, http.MethodPut, http.MethodTrace}

type HTTPCase struct {
	*integrate.TestCase
}

func NewHTTPCase(t *testing.T, serverProto, meshProto types.ProtocolName, server util.UpstreamServer) *HTTPCase {
	c := integrate.NewTestCase(t, serverProto, meshProto, server)
	return &HTTPCase{c}
}

func (c *HTTPCase) RunCase(n int, interval int) {
	client := http.DefaultClient
	if c.AppProtocol == protocol.HTTP2 {
		tr := &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(netw, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(netw, addr)
			},
		}
		client = &http.Client{Transport: tr}
	}
	c.runCaseWithClient(client, n, interval)
}

func (c *HTTPCase) runCaseWithClient(client *http.Client, n, interval int) {
	callHttp := func(req *http.Request) error {
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("response status: %d", resp.StatusCode)
		}
		if _, err := ioutil.ReadAll(resp.Body); err != nil {
			return err
		}
		return nil
	}
	call := func() error {
		for _, m := range allMethods {
			req, err := http.NewRequest(m, fmt.Sprintf("http://%s/%s", c.ClientMeshAddr, m), nil)
			if err != nil {
				return errorWrapper{method: m, original: err}
			}
			if err := callHttp(req); err != nil {
				return errorWrapper{method: m, original: err}
			}
		}
		return nil
	}
	for i := 0; i < n; i++ {
		if err := call(); err != nil {
			c.C <- err
			return
		}
		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
	c.C <- nil
}

func TestHTTPMethod(t *testing.T) {
	appaddr := "127.0.0.1:8080"
	// support non-tls/tls/proxy mode
	for _, f := range []func(c *HTTPCase){
		func(c *HTTPCase) {
			c.Start(false)
		},
		func(c *HTTPCase) {
			c.Start(true)
		},
		func(c *HTTPCase) {
			c.StartProxy()
		},
	} {
		testCases := []*HTTPCase{
			NewHTTPCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, &MethodHTTPHandler{})),
			// NewHTTPCase(t, protocol.HTTP1, protocol.HTTP2, util.NewHTTPServer(t, &MethodHTTPHandler{})),
			NewHTTPCase(t, protocol.HTTP2, protocol.HTTP2, util.NewUpstreamHTTP2(t, appaddr, &MethodHTTPHandler{})),
			// NewHTTPCase(t, protocol.HTTP2, protocol.HTTP1, util.NewUpstreamHTTP2(t, appaddr, &MethodHTTPHandler{})),
		}
		for i, tc := range testCases {
			t.Logf("start case #%d\n", i)
			f(tc)
			go tc.RunCase(1, 0)
			select {
			case err := <-tc.C:
				if err != nil {
					t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v test failed, error: %v\n", i, tc.AppProtocol, tc.MeshProtocol, err)
				}
			case <-time.After(15 * time.Second):
				t.Errorf("[ERROR MESSAGE] #%d %v to mesh %v hang\n", i, tc.AppProtocol, tc.MeshProtocol)
			}
			tc.FinishCase()
		}
	}
}

func TestHttp1DownstreamReset(t *testing.T) {
	handler := &MethodHTTPHandler{}
	c := NewHTTPCase(t, protocol.HTTP1, protocol.HTTP1, util.NewHTTPServer(t, handler))
	c.StartProxy()

	for i := 0; i < 2; i++ {
		client := http.Client{}
		if i == 0 {
			handler.latency = func() time.Duration {
				return time.Second
			}
			client.Timeout = time.Millisecond * 500
		} else {
			handler.latency = nil
		}
		go c.runCaseWithClient(&client, 1, 0)
		select {
		case err := <-c.C:
			if err != nil {
				if err, ok := err.(errorWrapper); ok && err.Timeout() {
					t.Logf("request timeout, error:%v", err)
				} else {
					t.Errorf("[ERROR MESSAGE] test failed, error: %v", err)
				}
			}
		case <-time.After(15 * time.Second):
			t.Errorf("[ERROR MESSAGE] hang\n")
		}
	}
	c.FinishCase()
}
