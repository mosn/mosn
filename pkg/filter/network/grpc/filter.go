package grpc

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/pkg/buffer"
)

type grpcFilter struct {
	ln   *Listener
	conn *Connection
}

var _ api.ReadFilter = &grpcFilter{}

// TODO: maybe we needs the context in the future
func NewGrpcFilter(_ context.Context, ln *Listener) *grpcFilter {
	return &grpcFilter{
		ln: ln,
	}
}

func (f *grpcFilter) OnData(buf buffer.IoBuffer) api.FilterStatus {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc filter received a data buffer: %d", buf.Len())
	}
	f.dispatch(buf)
	// grpc filer should be the final network filter.
	return api.Stop
}

func (f *grpcFilter) OnNewConnection() api.FilterStatus {
	// send a grpc connection to Listener to awake Listener's Accept
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc filter send a new connection to grpc server")
	}
	f.ln.NewConnection(f.conn)
	return api.Continue
}

func (f *grpcFilter) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("grpc filter received a new connection: %v", cb.Connection())
	}
	conn := cb.Connection()
	f.conn = NewConn(conn)
}

func (f *grpcFilter) dispatch(buf buffer.IoBuffer) {
	for buf.Len() > 0 {
		if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
			log.DefaultLogger.Debugf("grpc get datas: %d", buf.Len())
		}
		// send data to awake connection Reead
		f.conn.Send(buf)
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("read dispatch finished")
	}
}
