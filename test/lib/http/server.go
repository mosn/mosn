package http

import (
	"net"
	"net/http"
	"sync"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/test/lib"
	"mosn.io/mosn/test/lib/types"
)

func init() {
	lib.RegisterCreateServer("Http1", NewHTTPServer)
}

type MockHttpServer struct {
	mutex sync.Mutex
	addr  string
	mux   map[string]func(http.ResponseWriter, *http.Request)
	stats *types.ServerStats

	// running state
	server   *http.Server
	listener *StatsListener
}

func NewHTTPServer(config interface{}) types.MockServer {
	cfg, err := NewHttpServerConfig(config)
	if err != nil {
		log.DefaultLogger.Errorf("new http server config error: %v", err)
		return nil
	}
	if len(cfg.Configs) == 0 { // use default config
		cfg.Configs = map[string]*ResonseConfig{
			"/": &ResonseConfig{
				Condition: nil, // always matched true
				CommonBuilder: &ResponseBuilder{
					StatusCode: http.StatusOK,
					Header: map[string][]string{
						"mosn-test-default": []string{"http1"},
					},
					Body: string("default-http1"),
				},
				ErrorBuilder: &ResponseBuilder{
					StatusCode: http.StatusInternalServerError, // 500
					Body:       string("condition is not matched"),
				},
			},
		}

	}
	s := &MockHttpServer{
		addr:   cfg.Addr,
		mux:    make(map[string]func(http.ResponseWriter, *http.Request)),
		stats:  types.NewServerStats(),
		server: &http.Server{},
	}
	for pattern, c := range cfg.Configs {
		s.mux[pattern] = c.ServeHTTP
	}
	return s
}

// Wrap a http server so we can get the metrics info
func (s *MockHttpServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// receive a request
	s.stats.Records().RecordRequest()
	ww := &ResponseWriterWrapper{
		ResponseWriter: w,
		stats:          s.stats,
	}
	handler, ok := s.mux[r.URL.Path]
	if !ok {
		ww.WriteHeader(http.StatusNotFound)
		return
	}
	handler(ww, r)
}

func (s *MockHttpServer) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.listener != nil {
		return
	}
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		log.DefaultLogger.Fatalf("listen %s failed", s.addr)
	}
	s.listener = NewStatsListener(ln, s.stats)

	// register http server
	mux := &http.ServeMux{}
	mux.HandleFunc("/", s.ServeHTTP) // use mock server http
	s.server.Handler = mux
	go s.server.Serve(s.listener)

}

func (s *MockHttpServer) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.server.Close()
	s.listener = nil
}

func (s *MockHttpServer) Stats() types.ServerStatsReadOnly {
	return s.stats
}
