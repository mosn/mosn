package http

import (
	"testing"
	"time"
)

func TestHttp2Simple(t *testing.T) {
	config := &HttpServerConfig{
		Addr: "127.0.0.1:8080",
	}
	srv := NewHTTP2Server(config)
	srv.Start()
	defer func() {
		srv.Stop()
	}()

	time.Sleep(time.Second) // wait server start
	client := NewHttp2Client(&HttpClientConfig{
		TargetAddr: "127.0.0.1:8080",
		MaxConn:    1,
		Request:    nil,
		Verify: &VerifyConfig{
			ExpectedStatusCode: 200,
			ExpectedHeader: map[string][]string{
				"mosn-test-default": []string{"http1"},
			},
			ExpectedBody: []byte("default-http1"),
		},
	})
	for i := 0; i < 10; i++ {
		if !client.SyncCall() {
			t.Fatal("request failed")
		}
	}
	// verify stats
	stats := srv.Stats()
	connTotal := stats.ConnectionTotal()
	connActive := stats.ConnectionActive()
	respInfo, total := stats.ResponseInfo()
	request := stats.Requests()
	if !(connTotal == 1 &&
		connActive == 1 &&
		respInfo[200] == 10 &&
		total == 10 &&
		request == 10) {
		t.Fatalf("server metrics not expected: %v", stats)
	}
	cstats := client.Stats()
	crespInfo, ctotal := cstats.ResponseInfo()
	if !(cstats.ConnectionTotal() == 1 &&
		cstats.ConnectionActive() == 1 &&
		cstats.Requests() == 10 &&
		cstats.ExpectedResponseCount() == 10 &&
		crespInfo[200] == 10 &&
		ctotal == 10) {
		t.Fatalf("client metrics not expected: %v", cstats)
	}
	// close connection
	client.Close()
	time.Sleep(time.Second) // wait connection closed event
	if !(cstats.ConnectionActive() == 0 &&
		cstats.ConnectionTotal() == 1 &&
		cstats.ConnectionClosed() == 1 &&
		stats.ConnectionActive() == 0 &&
		stats.ConnectionClosed() == 1 &&
		stats.ConnectionTotal() == 1) {
		t.Fatalf("conn metrics not expected: %v, %v", cstats, stats)
	}
}
