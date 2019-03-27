package transport

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/TarsCloud/TarsGo/tars/util/rogger"
	"github.com/TarsCloud/TarsGo/tars/util/rtimer"
)

const (
	//PACKAGE_LESS shows is not a completed package.
	PACKAGE_LESS = iota
	//PACKAGE_FULL shows is a completed package.
	PACKAGE_FULL
	//PACKAGE_ERROR shows is a error package.
	PACKAGE_ERROR
)

//TLOG  is logger for transport.
var TLOG = rogger.GetLogger("TLOG")

//TarsProtoCol is interface for handling the server side tars package.
type TarsProtoCol interface {
	Invoke(ctx context.Context, pkg []byte) []byte
	ParsePackage(buff []byte) (int, int)
	InvokeTimeout(pkg []byte) []byte
}

//ServerHandler  is interface with listen and handler method
type ServerHandler interface {
	Listen() error
	Handle() error
}

//TarsServerConf server config for tars server side.
type TarsServerConf struct {
	Proto          string
	Address        string
	MaxInvoke      int32
	AcceptTimeout  time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	HandleTimeout  time.Duration
	IdleTimeout    time.Duration
	QueueCap       int
	TCPReadBuffer  int
	TCPWriteBuffer int
	TCPNoDelay     bool
}

//TarsServer tars server struct.
type TarsServer struct {
	svr        TarsProtoCol
	conf       *TarsServerConf
	lastInvoke time.Time
	idleTime   time.Time
	isClosed   bool
	numInvoke  int32
}

//NewTarsServer new TarsServer and init with conf.
func NewTarsServer(svr TarsProtoCol, conf *TarsServerConf) *TarsServer {
	ts := &TarsServer{svr: svr, conf: conf}
	ts.isClosed = false
	ts.lastInvoke = time.Now()
	return ts
}

func (ts *TarsServer) getHandler() (sh ServerHandler) {
	if ts.conf.Proto == "tcp" {
		sh = &tcpHandler{conf: ts.conf, ts: ts}
	} else if ts.conf.Proto == "udp" {
		sh = &udpHandler{conf: ts.conf, ts: ts}
	} else {
		panic("unsupport protocol: " + ts.conf.Proto)
	}
	return
}

//Serve listen and handle
func (ts *TarsServer) Serve() error {
	h := ts.getHandler()
	if err := h.Listen(); err != nil {
		return err
	}
	return h.Handle()
}

//Shutdown shutdown the server.
func (ts *TarsServer) Shutdown() {
	ts.isClosed = true
}

//GetConfig gets the tars server config.
func (ts *TarsServer) GetConfig() *TarsServerConf {
	return ts.conf
}

//IsZombie show whether the server is hanged by the request.
func (ts *TarsServer) IsZombie(timeout time.Duration) bool {
	conf := ts.GetConfig()
	return conf.MaxInvoke != 0 && ts.numInvoke == conf.MaxInvoke && ts.lastInvoke.Add(timeout).Before(time.Now())
}

func (ts *TarsServer) invoke(ctx context.Context, pkg []byte) []byte {
	cfg := ts.conf
	atomic.AddInt32(&ts.numInvoke, 1)
	var rsp []byte
	if cfg.HandleTimeout == 0 {
		rsp = ts.svr.Invoke(ctx, pkg)
	} else {
		done := make(chan struct{})
		go func() {
			rsp = ts.svr.Invoke(ctx, pkg)
			done <- struct{}{}
		}()
		select {
		case <-rtimer.After(cfg.HandleTimeout):
			rsp = ts.svr.InvokeTimeout(pkg)
		case <-done:
		}
	}
	atomic.AddInt32(&ts.numInvoke, -1)
	return rsp
}
