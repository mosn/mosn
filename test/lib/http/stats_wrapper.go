package http

import (
	"net"
	"net/http"

	"mosn.io/mosn/test/lib/types"
)

// StatsListener wraps a listener so we can set the metrics
type StatsListener struct {
	listener net.Listener
	stats    *types.ServerStats
}

func NewStatsListener(ln net.Listener, stats *types.ServerStats) *StatsListener {
	return &StatsListener{
		listener: ln,
		stats:    stats,
	}
}

func (sl *StatsListener) Accept() (net.Conn, error) {
	conn, err := sl.listener.Accept()
	if err != nil {
		return conn, err
	}
	sl.stats.ActiveConnection()
	statsConn := &StatsConn{
		Conn: conn,
		CloseEvent: func() {
			sl.stats.CloseConnection()
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

// StatsConn wraps a connection  so we can set the metrics
type StatsConn struct {
	net.Conn
	CloseEvent func()
}

func (conn *StatsConn) Close() error {
	err := conn.Conn.Close()
	if err == nil {
		conn.CloseEvent()
	}
	return err
}

// ResponseWriterWrapper wraps a response so we can set the metrics
type ResponseWriterWrapper struct {
	http.ResponseWriter
	stats *types.ServerStats
}

func (w *ResponseWriterWrapper) WriteHeader(code int) {
	w.stats.Records().RecordResponse(int16(code))
	w.ResponseWriter.WriteHeader(code)
}
