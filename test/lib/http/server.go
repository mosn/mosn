package http

import (
	"net"
	"net/http"
	"sync"
)

// StatsListener implements a net.Listener for http.Server
type StatsListener struct {
	listener net.Listener
	// stats
	*ConnStats
}

func NewStatsListener(ln net.Listener) *StatsListener {
	return &StatsListener{
		listener:  ln,
		ConnStats: &ConnStats{},
	}
}

func (sl *StatsListener) Accept() (net.Conn, error) {
	conn, err := sl.listener.Accept()
	if err != nil {
		return conn, err
	}
	sl.ConnStats.ActiveConnection()
	statsConn := &StatsConn{
		Conn: conn,
		cb: func() {
			sl.ConnStats.CloseConnection()
		},
	}
	return statsConn, nil
}

func (sl *StatsListener) Close() error {
	return sl.listener.Close()
}

func (sl *StatsListener) Addr() net.Addr {
	return sl.listener.Addr()
}

type StatsConn struct {
	net.Conn
	cb func()
}

func (conn *StatsConn) Close() error {
	err := conn.Conn.Close()
	if err == nil {
		conn.cb()
	}
	return err
}

type MockServer struct {
	*http.Server
	*ServerStats
	Addr     string
	Mux      map[string]func(http.ResponseWriter, *http.Request)
	listener *StatsListener
	lock     sync.Mutex
}

func NewMockServer(addr string, f ServeFunc) *MockServer {
	srv := &MockServer{
		Server: &http.Server{},
		Addr:   addr,
		Mux:    make(map[string]func(http.ResponseWriter, *http.Request)),
		lock:   sync.Mutex{},
		// Stats and listener in init in Start()
	}
	// Register Mux
	if f == nil {
		f = DefaultHTTPServe.Serve
	}
	f(srv)
	return srv
}

func (srv *MockServer) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	srv.Mux[pattern] = handler
}

type ResponseWriterWrapper struct {
	http.ResponseWriter
	stats *ServerStats
}

func (w *ResponseWriterWrapper) WriteHeader(code int) {
	w.stats.Response(int16(code))
	w.ResponseWriter.WriteHeader(code)
}

// A wrapper of ServerHTTP
func (srv *MockServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.ServerStats.Request()
	ww := &ResponseWriterWrapper{
		ResponseWriter: w,
		stats:          srv.ServerStats,
	}
	// path
	handler, ok := srv.Mux[r.URL.Path]
	if !ok {
		ww.WriteHeader(http.StatusNotFound)
		return
	}
	handler(ww, r)
}

func (srv *MockServer) Start() {
	srv.lock.Lock()
	if srv.listener != nil {
		srv.lock.Unlock()
		return
	}
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		panic(err)
	}
	srv.listener = NewStatsListener(ln)
	srv.ServerStats = NewServerStats(srv.listener.ConnStats)
	srv.lock.Unlock()
	// register http server
	mux := &http.ServeMux{}
	mux.HandleFunc("/", srv.ServeHTTP)
	srv.Handler = mux
	srv.Serve(srv.listener)
}
