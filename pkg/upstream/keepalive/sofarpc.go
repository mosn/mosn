package keepalive

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc"
	_ "github.com/alipay/sofa-mosn/pkg/protocol/rpc/sofarpc/codec"
	"github.com/alipay/sofa-mosn/pkg/stream"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// StreamReceiver to receive keep alive response
type sofaRPCKeepAlive struct {
	Codec        stream.CodecClient
	ProtocolByte byte
	Timeout      time.Duration
	Threshold    uint32
	Callbacks    []types.KeepAliveCallback
	// runtime
	timeoutCount uint32
	try          chan bool
	// requests records all running request
	// a request is handled once: response or timeout
	requests map[uint64]*keepAliveTimeout
	mutex    sync.Mutex
}

func NewSofaRPCKeepAlive(codec stream.CodecClient, proto byte, timeout time.Duration, thres uint32) types.KeepAlive {
	kp := &sofaRPCKeepAlive{
		Codec:        codec,
		ProtocolByte: proto,
		Timeout:      timeout,
		Threshold:    thres,
		Callbacks:    []types.KeepAliveCallback{},
		timeoutCount: 0,
		try:          make(chan bool, thres),
		requests:     make(map[uint64]*keepAliveTimeout),
		mutex:        sync.Mutex{},
	}
	return kp
}

func (kp *sofaRPCKeepAlive) Start() {
	for {
		select {
		case <-kp.try:
			kp.sendKeepAlive()
			// TODO: other action
		}
	}
}

func (kp *sofaRPCKeepAlive) AddCallback(cb types.KeepAliveCallback) {
	kp.Callbacks = append(kp.Callbacks, cb)
}

func (kp *sofaRPCKeepAlive) runCallback(status types.KeepAliveStatus) {
	for _, cb := range kp.Callbacks {
		cb(status)
	}
}

// SendKeepAlive will make a request to server via codec.
// use channel, do not block
func (kp *sofaRPCKeepAlive) SendKeepAlive() {
	select {
	case kp.try <- true:
	default:
		log.DefaultLogger.Warnf("keep alive too much")
	}
}

// The function will be called when connection in the codec is idle
func (kp *sofaRPCKeepAlive) sendKeepAlive() {
	ctx := context.Background()
	sender := kp.Codec.NewStream(ctx, kp)
	id := sender.GetStream().ID()
	// we send sofa rpc cmd as "header", but it maybe contains "body"
	hb := sofarpc.NewHeartbeat(kp.ProtocolByte)
	sender.AppendHeaders(ctx, hb, true)
	// start a timer for request
	kp.mutex.Lock()
	kp.requests[id] = startTimeout(id, kp)
	kp.mutex.Unlock()
}

func (kp *sofaRPCKeepAlive) GetTimeout() time.Duration {
	return kp.Timeout
}

func (kp *sofaRPCKeepAlive) HandleTimeout(id uint64) {
	kp.mutex.Lock()
	defer kp.mutex.Unlock()
	if _, ok := kp.requests[id]; ok {
		delete(kp.requests, id)
		atomic.AddUint32(&kp.timeoutCount, 1)
		// close the connection
		if kp.timeoutCount > kp.Threshold {
			kp.Codec.Close()
		}
		kp.runCallback(types.KeepAliveTimeout)
	}
}

func (kp *sofaRPCKeepAlive) HandleSuccess(id uint64) {
	kp.mutex.Lock()
	defer kp.mutex.Unlock()
	if timeout, ok := kp.requests[id]; ok {
		delete(kp.requests, id)
		timeout.timer.stop()
		// reset the tiemout count
		atomic.StoreUint32(&kp.timeoutCount, 0)
		kp.runCallback(types.KeepAliveSuccess)
	}
}

// StreamReceiver Implementation
// we just needs to make sure we can receive a response, do not care the data we received
func (kp *sofaRPCKeepAlive) OnReceiveHeaders(ctx context.Context, headers types.HeaderMap, endStream bool) {
	if ack, ok := headers.(sofarpc.SofaRpcCmd); ok {
		kp.HandleSuccess(ack.RequestID())
	}
}

func (kp *sofaRPCKeepAlive) OnReceiveData(ctx context.Context, data types.IoBuffer, endStream bool) {
	// ignore
	// TODO: release the iobuffer
}

func (kp *sofaRPCKeepAlive) OnReceiveTrailers(ctx context.Context, trailers types.HeaderMap) {
}

func (kp *sofaRPCKeepAlive) OnDecodeError(ctx context.Context, err error, headers types.HeaderMap) {
}

//
type keepAliveTimeout struct {
	ID        uint64
	timer     *timer
	KeepAlive types.KeepAlive
}

func startTimeout(id uint64, keep types.KeepAlive) *keepAliveTimeout {
	t := &keepAliveTimeout{
		ID:        id,
		KeepAlive: keep,
	}
	t.timer = newTimer(t.onTimeout)
	t.timer.start(keep.GetTimeout())
	return t
}

func (t *keepAliveTimeout) onTimeout() {
	t.KeepAlive.HandleTimeout(t.ID)
}
