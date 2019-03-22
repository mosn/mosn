package tars

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/endpointf"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	"github.com/TarsCloud/TarsGo/tars/transport"
	"sync"
	"sync/atomic"
	"time"
)

// AdapterProxy : Adapter proxy
type AdapterProxy struct {
	resp       sync.Map
	point      *endpointf.EndpointF
	tarsClient *transport.TarsClient
	comm       *Communicator
	failCount  int32
	sendCount  int32
	status     bool
	closed     bool
}

// New : Construct an adapter proxy
func (c *AdapterProxy) New(point *endpointf.EndpointF, comm *Communicator) error {
	c.comm = comm
	c.point = point
	proto := "tcp"
	if point.Istcp == 0 {
		proto = "udp"
	}

	conf := &transport.TarsClientConf{
		Proto: proto,
		//NumConnect:   netthread,
		QueueLen:     ClientQueueLen,
		IdleTimeout:  ClientIdleTimeout,
		ReadTimeout:  ClientReadTimeout,
		WriteTimeout: ClientWriteTimeout,
	}
	c.tarsClient = transport.NewTarsClient(fmt.Sprintf("%s:%d", point.Host, point.Port), c, conf)
	c.status = true
	go c.checkActive()
	return nil
}

// ParsePackage : Parse packet from bytes
func (c *AdapterProxy) ParsePackage(buff []byte) (int, int) {
	return TarsRequest(buff)
}

// Recv : Recover read channel when closed for timeout
func (c *AdapterProxy) Recv(pkg []byte) {
	defer func() {
		// TODO readCh has a certain probability to be closed after the load, and we need to recover
		// Maybe there is a better way
		if err := recover(); err != nil {
			TLOG.Error("recv pkg painc:", err)
		}
	}()
	packet := requestf.ResponsePacket{}
	err := packet.ReadFrom(codec.NewReader(pkg))
	if err != nil {
		TLOG.Error("decode packet error", err.Error())
		return
	}
	chIF, ok := c.resp.Load(packet.IRequestId)
	if ok {
		ch := chIF.(chan *requestf.ResponsePacket)
		TLOG.Debug("IN:", packet)
		ch <- &packet
	} else {
		TLOG.Error("timeout resp,drop it:", packet.IRequestId)
	}
}

// Send : Send packet
func (c *AdapterProxy) Send(req *requestf.RequestPacket) error {
	TLOG.Debug("send req:", req.IRequestId)
	c.sendAdd()
	sbuf := bytes.NewBuffer(nil)
	sbuf.Write(make([]byte, 4))
	os := codec.NewBuffer()
	req.WriteTo(os)
	bs := os.ToBytes()
	sbuf.Write(bs)
	len := sbuf.Len()
	binary.BigEndian.PutUint32(sbuf.Bytes(), uint32(len))
	return c.tarsClient.Send(sbuf.Bytes())
}

// GetPoint : Get an endpoint
func (c *AdapterProxy) GetPoint() *endpointf.EndpointF {
	return c.point
}

// Close : Close the client
func (c *AdapterProxy) Close() {
	c.tarsClient.Close()
	c.closed = true
}

func (c *AdapterProxy) sendAdd() {
	atomic.AddInt32(&c.sendCount, 1)
}

func (c *AdapterProxy) failAdd() {
	atomic.AddInt32(&c.failCount, 1)
}

func (c *AdapterProxy) reset() {
	atomic.SwapInt32(&c.sendCount, 0)
	atomic.SwapInt32(&c.failCount, 0)
}

func (c *AdapterProxy) checkActive() {
	loop := time.NewTicker(AdapterProxyTicker)
	count := 0 // Detect if a dead node recovers each minute
	for range loop.C {
		if c.closed {
			loop.Stop()
			return
		}
		if c.failCount > c.sendCount/2 {
			c.status = false
		}
		if !c.status && count > AdapterProxyResetCount {
			//TODO USE TAFPING INSTEAD
			c.reset()
			c.status = true
			count = 0
		}
		count++
	}
}
