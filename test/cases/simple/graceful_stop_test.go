//go:build MOSNTest
// +build MOSNTest

package simple

import (
	"bytes"
	"context"
	"crypto/tls"
	"io/ioutil"
	"net"
	goHttp "net/http"
	"testing"
	"time"

	"golang.org/x/net/http2"
	"mosn.io/mosn/pkg/module/http2/h2c"
	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
)

var body = make([]byte, 1<<18)

func TestHttp2NotUseStream2(t *testing.T) {
	Scenario(t, "http2 not use stream", func() {
		var m *mosn.MosnOperator
		var server *goHttp.Server
		Setup(func() {
			m = mosn.StartMosn(ConfigSimpleHTTP2)
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("client-mosn-server", func() {
			go func() {
				h2s := &http2.Server{}
				handler := goHttp.HandlerFunc(func(w goHttp.ResponseWriter, r *goHttp.Request) {
					var body []byte
					var err error
					if body, err = ioutil.ReadAll(r.Body); err != nil {
						t.Fatalf("error request body %v", err)
					}
					w.Header().Set("Content-Type", "text/plain")
					time.Sleep(time.Millisecond * 500)
					w.Write(body)
				})
				server = &goHttp.Server{
					Addr:    "0.0.0.0:8080",
					Handler: h2c.NewHandler(handler, h2s),
				}
				_ = server.ListenAndServe()
			}()

			time.Sleep(time.Second)

			testcases := []struct {
				reqBody []byte
			}{
				{
					reqBody: []byte("xxxxx"),
				},
			}

			for _, tc := range testcases {
				client := goHttp.Client{
					Transport: &http2.Transport{
						AllowHTTP: true,
						DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
							return net.Dial(network, addr)
						},
					},
				}
				// 1. simple
				resp, err := client.Post("http://localhost:2046", "text/plain", bytes.NewReader(tc.reqBody))
				Verify(err, Equal, nil)
				respBody, err := ioutil.ReadAll(resp.Body)
				Verify(err, Equal, nil)
				Verify(len(respBody), Equal, len(tc.reqBody))

				// 2. graceful stop after send request
				resp, err = client.Post("http://localhost:2046", "text/plain", bytes.NewReader(tc.reqBody))
				m.GracefulStop()
				Verify(err, Equal, nil)
				respBody, err = ioutil.ReadAll(resp.Body)
				Verify(err, Equal, nil)
				Verify(len(respBody), Equal, len(tc.reqBody))
			}
		})
		TearDown(func() {
			m.Stop()
			_ = server.Shutdown(context.TODO())
		})
	})
}
