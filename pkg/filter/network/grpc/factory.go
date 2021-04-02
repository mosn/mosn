package grpc

import (
	"context"
	"errors"
	"net"
	"sync"

	"google.golang.org/grpc"
	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

func init() {
	api.RegisterNetwork(v2.GRPC, CreateGRPCServerFilterFactory)
}

// grpcServerFilterFactory contains a protobuf registered grpc server that provides grpc service
// The registered grpc server needs to be implemented independently by the user, and registered to factory.
// A MOSN can contains multiple registered grpc servers, each grpc server has only one instance.
type grpcServerFilterFactory struct {
	server *grpc.Server
	ln     *Listener
}

var _ api.NetworkFilterChainFactory = &grpcServerFilterFactory{}

func (f *grpcServerFilterFactory) CreateFilterChain(ctx context.Context, callbacks api.NetWorkFilterChainFactoryCallbacks) {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("create a new grpc filter")
	}
	rf := NewGrpcFilter(ctx, f.ln)
	callbacks.AddReadFilter(rf)
}

var (
	ErrUnknownServer    = errors.New("cannot found the grpc server")
	ErrDuplicateStarted = errors.New("grpc server has already started")
)

func CreateGRPCServerFilterFactory(conf map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	name := conf["server_name"]
	handler := getRegisterServerHandler(name)
	if handler == nil {
		log.DefaultLogger.Errorf("invalid grpc server found: %s", name)
		return nil, ErrUnknownServer
	}
	// TODO: Close should be called when filter's listener is closed
	ln := NewListener(conf)
	// TODO: support dynamic upgrade
	srv, ok := handler.Start(ln)
	if !ok {
		log.DefaultLogger.Errorf("mosn does not support repeated start grpc server")
		return nil, ErrDuplicateStarted
	}
	log.DefaultLogger.Debugf("grpc server filter: create a grpc server name: %s", name)
	return &grpcServerFilterFactory{
		server: srv,
		ln:     ln,
	}, nil
}

// NewRegisteredServer returns a grpc server that has completed protobuf registration.
// The grpc server Serve function should be called in MOSN.
// NOTICE: some grpc options is not supported, for example, secure options, which are managed by MOSN too.
type NewRegisteredServer func() *grpc.Server

// Handler is a wrapper to call NewRegisteredServer
type Handler struct {
	once sync.Once
	f    NewRegisteredServer
}

// Start call the grpc server Serves. If the server is already started, ok flags returns false
func (s *Handler) Start(ln net.Listener) (srv *grpc.Server, ok bool) {
	s.once.Do(func() {
		srv = s.f()
		go func() {
			// TODO: support restart
			if err := srv.Serve(ln); err != nil {
				log.DefaultLogger.Errorf("grpc server serves error: %v", err)
			}
		}()
		ok = true
	})
	return srv, ok
}

var stores sync.Map

func RegisterServerHandler(name string, f NewRegisteredServer) bool {
	_, loaded := stores.LoadOrStore(name, &Handler{
		f: f,
	})
	// loaded means duplicate name registered
	ok := !loaded
	log.DefaultLogger.Infof("register a grpc server named: %s, success: %t", name, ok)
	return ok
}

func getRegisterServerHandler(name interface{}) *Handler {
	v, _ := stores.Load(name)
	h, ok := v.(*Handler)
	if !ok {
		return nil
	}
	return h
}
