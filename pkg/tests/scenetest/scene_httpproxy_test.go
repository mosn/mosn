package tests

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/alipay/sofamosn/pkg/mosn"
)

func GetServerAddr(s *httptest.Server) string {
	return strings.Split(s.URL, "http://")[1]
}

func ParseHttpResponse(t *testing.T, req *http.Request) string {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Errorf("request failed %v\n", req)
		return ""
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("read response body failed %v\n", req)
		return ""
	}
	re := regexp.MustCompile("\nServerName:[a-zA-Z0-9]+\n")
	return strings.Split(
		strings.Trim(re.FindString(string(body)), "\n"), ":",
	)[1]
}

func TestHttpProxy(t *testing.T) {
	//start httptest server
	cluster1Server := &HttpServer{
		t:    t,
		name: "server1",
	}
	server1 := httptest.NewServer(cluster1Server)
	defer server1.Close()
	cluster2Server := &HttpServer{
		t:    t,
		name: "server2",
	}
	server2 := httptest.NewServer(cluster2Server)
	defer server2.Close()
	//mesh config
	cluster1 := []string{GetServerAddr(server1)}
	cluster2 := []string{GetServerAddr(server2)}
	//	meshAddr := "127.0.0.1:2045"
	meshAddr := "127.0.0.1:2048"
	mesh_config := CreateHTTPRouteConfig(meshAddr, [][]string{cluster1, cluster2})
	go mosn.Start(mesh_config, "", "")
	time.Sleep(5 * time.Second) //wait mesh and server start
	makeRequest := func(header string, path string) *http.Request {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/%s", meshAddr, path), nil)
		if err != nil {
			t.Fatalf("create request error:%v\n", err)
		}
		req.Header.Add("service", header)
		return req
	}
	//cluster1
	if clustername := ParseHttpResponse(t, makeRequest("cluster1", "")); clustername != cluster1Server.name {
		t.Errorf("expected %s, but got %s\n", cluster1Server.name, clustername)
	}
	//cluster2
	if clustername := ParseHttpResponse(t, makeRequest("cluster2", "")); clustername != cluster2Server.name {
		t.Errorf("expected %s, but got %s\n", cluster2Server.name, clustername)
	}
	//cluster2 path
	if clustername := ParseHttpResponse(t, makeRequest("cluster1", "test.htm")); clustername != cluster2Server.name {
		t.Errorf("expected %s, but got %s\n", cluster2Server.name, clustername)
	}
}
