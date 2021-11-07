package integrate

import (
	"fmt"
	"net/http"
	"testing"

	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/test/lib/mosn"
	"mosn.io/mosn/test/util"
	"mosn.io/mosn/test/util/mosn"
)

func serveHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Connection", "upgrade")
	w.Header().Set("Upgrade", "websocket")

	r.Header.Write(w)
}

func startUpstreamServer(addr string) {
	http.HandleFunc("/", serveHTTP)
	http.ListenAndServe(addr, nil)
}

func TestWebsocketProxy(t *testing.T) {
	upAddr := "127.0.0.1:8080"
	listenerAddr := "127.0.0.1:8081"

	cfg := util.CreateProxyMesh(listenerAddr, []string{upAddr}, protocol.HTTP1)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()

	go startUpstreamServer(upAddr)

	resp, err := http.Get(fmt.Sprintf("http://%s/%s", listenerAddr, HTTPTestPath))
	if err != nil {
		t.Errorf("web socket proxy get resp failed: %v", err)
	}

	ch := resp.Header.Get("Connection")
	if ch != "upgrade" {
		t.Errorf("web socket proxy get resp failed, header \"Connection\" value:%s, \"upgrade\" expected", ch)
	}

	upgrade := resp.Header.Get("Upgrade")
	if upgrade != "websocket" {
		t.Errorf("web socket proxy get resp failed, header \"Upgrade\" value:%s, \"websocket\" expected", upgrade)
	}
}
