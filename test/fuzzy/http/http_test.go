package http

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"testing"
	"time"

	_ "github.com/alipay/sofa-mosn/pkg/filter/network/proxy"
	"github.com/alipay/sofa-mosn/pkg/log"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http"
	_ "github.com/alipay/sofa-mosn/pkg/stream/http2"
	_ "github.com/alipay/sofa-mosn/pkg/stream/sofarpc"
	"github.com/alipay/sofa-mosn/test/fuzzy"
	"github.com/alipay/sofa-mosn/test/util"
)

var (
	caseIndex    uint32 = 0
	caseDuration time.Duration
)

type HTTPClient struct {
	Client          *http.Client
	t               *testing.T
	url             string
	unexpectedCount uint32
	successCount    uint32
	failureCount    uint32
}

func NewHTTPClient(t *testing.T, addr string) *HTTPClient {
	httpClient := &http.Client{}
	return &HTTPClient{
		Client: httpClient,
		t:      t,
		url:    fmt.Sprintf("http://%s/", addr),
	}
}

func (c *HTTPClient) SendRequest() {
	resp, err := c.Client.Get(c.url)
	if err != nil {
		c.t.Errorf("unexpected error: %v\n", err)
		c.unexpectedCount++
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		c.successCount++
	} else {
		c.failureCount++
	}
	// Read Body
	ioutil.ReadAll(resp.Body)
}

type HTTPServer struct {
	server   *http.Server
	t        *testing.T
	ID       string
	mutex    sync.Mutex
	started  bool
	finished bool
}

func NewHTTPServer(t *testing.T, id string, addr string) *HTTPServer {
	s := &HTTPServer{
		t:     t,
		ID:    id,
		mutex: sync.Mutex{},
	}
	server := &http.Server{
		Handler: s,
		Addr:    addr,
	}
	s.server = server
	return s
}

func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for k := range r.Header {
		w.Header().Set(k, r.Header.Get(k))
	}
	fmt.Fprintf(w, "\nRequestId:%s\n", r.Header.Get("Requestid"))
}

//over write
func (s *HTTPServer) Close(finished bool) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	// once finished is set to true, it cannot be changed
	if !s.finished {
		s.finished = finished
	}
	if !s.started {
		return
	}
	log.StartLogger.Infof("[FUZZY TEST] server closed %s", s.ID)
	s.started = false
	s.server.Close()
}
func (s *HTTPServer) GoServe() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.started {
		return
	}
	s.started = true
	go s.server.ListenAndServe()
}
func (s *HTTPServer) ReStart() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.started {
		return
	}
	if s.finished {
		return
	}
	log.StartLogger.Infof("[FUZZY TEST] server restart #%s", s.ID)
	s.started = true
	go s.server.ListenAndServe()
}

func CreateServers(t *testing.T, serverList []string, stop chan struct{}) []fuzzy.Server {
	var servers []fuzzy.Server
	for i, s := range serverList {
		id := fmt.Sprintf("server#%d", i)
		server := NewHTTPServer(t, id, s)
		server.GoServe()
		go func(server *HTTPServer) {
			<-stop
			log.StartLogger.Infof("[FUZZY TEST] finished fuzzy server %s", server.ID)
			server.Close(true)
		}(server)
		servers = append(servers, server)
	}
	return servers
}

//main
func TestMain(m *testing.M) {
	util.MeshLogPath = "./logs/rpc.log"
	util.MeshLogLevel = "INFO"
	log.InitDefaultLogger(util.MeshLogPath, log.INFO)
	casetime := flag.Int64("casetime", 1, "-casetime=1(min)")
	flag.Parse()
	caseDuration = time.Duration(*casetime) * time.Minute
	log.StartLogger.Infof("each case at least run %v", caseDuration)
	m.Run()
}
