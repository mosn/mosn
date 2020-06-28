package transport

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"github.com/TarsCloud/TarsGo/tars/util/grace"
)

type udpHandler struct {
	conf *TarsServerConf
	ts   *TarsServer

	conn      *net.UDPConn
	numInvoke int32
}

func (h *udpHandler) Listen() (err error) {
	cfg := h.conf
	h.conn, err = grace.CreateUDPConn(cfg.Address)
	if err != nil {
		return err
	}
	TLOG.Info("UDP listen", h.conn.LocalAddr())
	return nil
}

func (h *udpHandler) Handle() error {
	atomic.AddInt32(&h.ts.numConn, 1)
	///wait invoke done
	defer func() {
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for atomic.LoadInt32(&h.ts.numInvoke) > 0 {
			select {
			case <-tick.C:
			}
		}
		atomic.AddInt32(&h.ts.numConn, -1)
	}()
	buffer := make([]byte, 65535)
	for {
		if atomic.LoadInt32(&h.ts.isClosed) == 1 {
			return nil
		}
		n, udpAddr, err := h.conn.ReadFromUDP(buffer)
		if err != nil {
			if atomic.LoadInt32(&h.ts.isClosed) == 1 {
				return nil
			}
			if isNoDataError(err) {
				continue
			} else {
				TLOG.Errorf("Close connection %s: %v", h.conf.Address, err)
				return err // TODO: check if necessary
			}
		}
		pkg := make([]byte, n)
		copy(pkg, buffer[0:n])
		go func() {
			ctx := context.Background()
			atomic.AddInt32(&h.ts.numInvoke, 1)
			rsp := h.ts.invoke(ctx, pkg[4:]) // no need to check package
			if _, err := h.conn.WriteToUDP(rsp, udpAddr); err != nil {
				TLOG.Errorf("send pkg to %v failed %v", udpAddr, err)
			}
			atomic.AddInt32(&h.ts.numInvoke, -1)
		}()
	}
	return nil
}

func (h *udpHandler) OnShutdown() {
}

func (h *udpHandler) CloseIdles(n int64) bool {
	if h.ts.numInvoke == 0 {
		h.conn.Close()
		return true
	}
	return false
}
