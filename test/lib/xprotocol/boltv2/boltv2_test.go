package boltv2

import (
	"testing"
	"time"
)

func TestBoltSimple(t *testing.T) {
	config := &BoltServerConfig{
		Addr: "127.0.0.1:8080",
	}
	srv := NewBoltServer(config)
	srv.Start()
	defer srv.Stop()

	time.Sleep(time.Second) // wait server start
	client := NewBoltClient(&BoltClientConfig{
		TargetAddr: "127.0.0.1:8080",
		MaxConn:    1,
		Verify: &VerifyConfig{
			ExpectedStatusCode: 0,
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
		respInfo[0] == 10 &&
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
		crespInfo[0] == 10 &&
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
