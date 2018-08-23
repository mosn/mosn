package network

import (
	"github.com/mailru/easygo/netpoll"
	"net"
	"sync"
	"errors"
	"log"
	"sync/atomic"
	mosnsync "github.com/alipay/sofa-mosn/pkg/sync"
)

var (
	// this pool if for event handle
	pool = mosnsync.NewSimplePool(16)

	rrCounter                   uint32 = 0
	poolSize                    uint32 = 1 //uint32(runtime.NumCPU())
	eventLoopPool                      = make([]*EventLoop, poolSize)
	eventAlreadyRegisteredError        = errors.New("event already registered")
)

func init() {
	for i := range eventLoopPool {
		poller, err := netpoll.New(nil)
		if err != nil {
			log.Fatalln("create poller failed, caused by ", err)
		}

		eventLoopPool[i] = &EventLoop{
			poller: poller,
			conn:   make(map[uint64]*ConnEvent), //TODO init size
		}
	}
}

func Attach() *EventLoop {
	return eventLoopPool[atomic.AddUint32(&rrCounter, 1)%poolSize]
}

type ConnEvent struct {
	read  *netpoll.Desc
	write *netpoll.Desc
}

type ConnEventHandler struct {
	OnHup   func() bool
	OnRead  func() bool
	OnWrite func() bool
}

type EventLoop struct {
	mu sync.Mutex

	poller netpoll.Poller

	conn map[uint64]*ConnEvent
}

func (el *EventLoop) Register(id uint64, conn net.Conn, handler *ConnEventHandler) error {
	// handle read
	read, err := netpoll.HandleReadOnce(conn)
	if err != nil {
		return err
	}

	// handle write
	write, err := netpoll.HandleWriteOnce(conn)
	if err != nil {
		return err
	}

	// register with wrapper
	el.poller.Start(read, el.readWrapper(read, handler))
	el.poller.Start(write, el.writeWrapper(write, handler))

	//store
	el.conn[id] = &ConnEvent{
		read:  read,
		write: write,
	}
	return nil
}

func (el *EventLoop) RegisterRead(id uint64, conn net.Conn, handler *ConnEventHandler) error {
	// handle read
	read, err := netpoll.HandleReadOnce(conn)
	if err != nil {
		return err
	}

	// register
	el.poller.Start(read, el.readWrapper(read, handler))

	el.mu.Lock()
	//store
	el.conn[id] = &ConnEvent{
		read: read,
	}
	el.mu.Unlock()

	return nil
}

func (el *EventLoop) RegisterWrite(id uint64, conn net.Conn, handler *ConnEventHandler) error {
	// handle write
	write, err := netpoll.HandleWriteOnce(conn)
	if err != nil {
		return err
	}

	// register
	el.poller.Start(write, el.writeWrapper(write, handler))

	//store
	el.conn[id] = &ConnEvent{
		write: write,
	}

	return nil
}

func (el *EventLoop) Unregister(id uint64) {

	if event, ok := el.conn[id]; ok {
		el.poller.Stop(event.read)
		el.poller.Stop(event.write)

		delete(el.conn, id)
	}

}

func (el *EventLoop) UnregisterRead(id uint64) {
	if event, ok := el.conn[id]; ok {
		el.poller.Stop(event.read)

		delete(el.conn, id)
	}
}

func (el *EventLoop) UnregisterWrite(id uint64) {
	if event, ok := el.conn[id]; ok {
		el.poller.Stop(event.write)

		delete(el.conn, id)
	}
}

func (el *EventLoop) readWrapper(desc *netpoll.Desc, handler *ConnEventHandler) func(netpoll.Event) {
	return func(e netpoll.Event) {
		// No more calls will be made for conn until we call epoll.Resume().
		if e&netpoll.EventReadHup != 0 {
			el.poller.Stop(desc)
			if !handler.OnHup() {
				return
			}
		}
		pool.Schedule(func() {
			if !handler.OnRead() {
				return
			}
			el.poller.Resume(desc)
		})
	}
}

func (el *EventLoop) writeWrapper(desc *netpoll.Desc, handler *ConnEventHandler) func(netpoll.Event) {
	return func(e netpoll.Event) {
		// No more calls will be made for conn until we call epoll.Resume().
		if e&netpoll.EventReadHup != 0 {
			el.poller.Stop(desc)
			if !handler.OnHup() {
				return
			}
		}
		pool.Schedule(func() {
			if !handler.OnWrite() {
				return
			}
			el.poller.Resume(desc)
		})
	}
}
