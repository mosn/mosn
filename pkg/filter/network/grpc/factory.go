/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package grpc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"runtime/debug"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/stagemanager"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/header"
	"mosn.io/pkg/variable"
)

func init() {
	api.RegisterNetwork(v2.GRPC_NETWORK_FILTER, CreateGRPCServerFilterFactory)
}

// grpcServerFilterFactory contains a protobuf registered grpc server that provides grpc service
// The registered grpc server needs to be implemented independently by the user, and registered to factory.
// A MOSN can contains multiple registered grpc servers, each grpc server has only one instance.
type grpcServerFilterFactory struct {
	config              *v2.GRPC
	handler             *Handler
	server              *registerServerWrapper
	streamFilterFactory streamfilter.StreamFilterFactory
}

var _ api.NetworkFilterChainFactory = (*grpcServerFilterFactory)(nil)
var _ api.FactoryInitializer = (*grpcServerFilterFactory)(nil)

func (f *grpcServerFilterFactory) CreateFilterChain(ctx context.Context, callbacks api.NetWorkFilterChainFactoryCallbacks) {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("create a new grpc filter")
	}
	rf := NewGrpcFilter(ctx, f.server.ln)
	callbacks.AddReadFilter(rf)
}

func (f *grpcServerFilterFactory) Init(param interface{}) error {
	var (
		sw  *registerServerWrapper
		err error
	)
	cfg, ok := param.(*v2.Listener)
	if !ok {
		return ErrInvalidConfig
	}
	addr := cfg.AddrConfig
	if addr == "" {
		addr = cfg.Addr.String()
	}

	// GetStreamFilters from listener name
	f.streamFilterFactory = streamfilter.GetStreamFilterManager().GetStreamFilterFactory(cfg.Name)

	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(f.UnaryInterceptorFilter),
		grpc.StreamInterceptor(f.StreamInterceptorFilter),
	}
	if cfg.Network == networkUnix {
		sw, err = f.handler.NewUnix(addr, f.config.GrpcConfig, opts...)
	} else {
		sw, err = f.handler.New(addr, f.config.GrpcConfig, opts...)
	}

	if err != nil {
		return err
	}
	// start server with timeout
	sw.Start(f.config.GracefulStopTimeout)
	f.server = sw
	log.DefaultLogger.Debugf("grpc server filter initialized success")
	return nil
}

// UnaryInterceptorFilter is an implementation of grpc.UnaryServerInterceptor, which used to be call stream filter in MOSN
func (f *grpcServerFilterFactory) UnaryInterceptorFilter(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	wr := &wrapper{}
	defer func() {
		// add recover, or process will be crashed if handler cause a panic
		if r := recover(); r != nil {
			log.DefaultLogger.Alertf(types.ErrorKeyProxyPanic, "[grpc] [unary] grpc unary handle panic: %v, method: %s, stack:%s", r, info.FullMethod, string(debug.Stack()))
		}
	}()
	sfc := streamfilter.GetDefaultStreamFilterChain()
	ss := &grpcStreamFilterChain{
		sfc,
		types.DownFilterAfterRoute,
		nil,
	}
	defer ss.destroy()

	f.streamFilterFactory.CreateFilterChain(ctx, ss)

	requestHeader := header.CommonHeader{}

	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		for k, v := range md {
			requestHeader.Set(k, v[0])
		}
	}

	ctx = variable.NewVariableContext(ctx)
	_ = variable.Set(ctx, types.VariableDownStreamProtocol, api.ProtocolName(grpcName))
	_ = variable.Set(ctx, types.VariableDownStreamReqHeaders, requestHeader)

	variable.SetString(ctx, VarGrpcServiceName, info.FullMethod)
	status := ss.RunReceiverFilter(ctx, api.AfterRoute, requestHeader, nil, nil, ss.receiverFilterStatusHandler)
	// when filter return StreamFiltertermination, should assign value to ss.err, Interceptor return directly
	if status == api.StreamFiltertermination {
		return nil, ss.err
	}

	if status == api.StreamFilterStop && ss.err != nil {
		err = ss.err
	} else {
		stream := grpc.ServerTransportStreamFromContext(ctx)
		wr = &wrapper{stream, nil, nil}
		newCtx := grpc.NewContextWithServerTransportStream(ctx, wr)
		//do biz logic
		resp, err = handler(newCtx, req)
	}

	responseHeader := header.CommonHeader{}
	for k, v := range wr.header {
		responseHeader.Set(k, v[0])
	}
	variable.Set(ctx, VarGrpcRequestResult, true)
	if err != nil {
		variable.Set(ctx, VarGrpcRequestResult, false)
	}
	responseTrailer := header.CommonHeader{}
	for k, v := range wr.trailer {
		responseTrailer.Set(k, v[0])
	}

	status = ss.RunSenderFilter(ctx, api.BeforeSend, responseHeader, nil, responseTrailer, ss.senderFilterStatusHandler)
	if status == api.StreamFiltertermination || status == api.StreamFilterStop {
		if err == nil {
			err = ss.err
		}
	}
	return
}

// StreamInterceptorFilter is an implementation of grpc.StreamServerInterceptor, which used to be call stream filter in MOSN
func (f *grpcServerFilterFactory) StreamInterceptorFilter(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	//TODO
	//to use mosn stream filter here, we must
	//1. only support intercept when the first request come and before the last response send (this maybe the standard implementation?)
	//2. support intercept every stream recv/send
	return handler(srv, ss)
}

var (
	ErrUnknownServer = errors.New("cannot found the grpc server")
	ErrInvalidConfig = errors.New("invalid config for grpc listener")
)

// CreateGRPCServerFilterFactory returns a grpc network filter factory.
// The conf contains more than v2.GRPC becasue some config is not ready yet. @nejisama
func CreateGRPCServerFilterFactory(conf map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	cfg, err := ParseGRPC(conf)
	if err != nil {
		log.DefaultLogger.Errorf("invalid grpc server config: %v, error: %v", conf, err)
		return nil, err
	}
	name := cfg.ServerName
	handler := getRegisterServerHandler(name)
	if handler == nil {
		log.DefaultLogger.Errorf("invalid grpc server found: %s", name)
		return nil, ErrUnknownServer
	}
	log.DefaultLogger.Debugf("grpc server filter: create a grpc server name: %s", name)
	return &grpcServerFilterFactory{
		config:  cfg,
		handler: handler,
	}, nil
}

// RegisteredServer is a wrapper of *(google.golang.org/grpc).Server
type RegisteredServer interface {
	Serve(net.Listener) error
	// GracefulStop graceful stop grpc server
	GracefulStop()
	// Stop ungraceful stop grpc server
	Stop()
}

// NewRegisteredServer returns a grpc server that has completed protobuf registration.
// The grpc server Serve function should be called in MOSN.
// NOTICE: some grpc options is not supported, for example, secure options, which are managed by MOSN too.
type NewRegisteredServer func(config json.RawMessage, options ...grpc.ServerOption) (RegisteredServer, error)

// Handler is a wrapper to call NewRegisteredServer
type Handler struct {
	f       NewRegisteredServer
	mutex   sync.Mutex
	servers map[string]*registerServerWrapper // support different listener start different grpc server
}

// New a grpc server with address. Same address returns same server, which can be start only once.
func (s *Handler) New(addr string, conf json.RawMessage, options ...grpc.ServerOption) (*registerServerWrapper, error) {
	return s.newGRPCServer(addr, networkTcp, conf, options...)
}

func (s *Handler) NewUnix(addr string, conf json.RawMessage, options ...grpc.ServerOption) (*registerServerWrapper, error) {
	return s.newGRPCServer(addr, networkUnix, conf, options...)
}

func (s *Handler) newGRPCServer(addr string, network string, conf json.RawMessage, options ...grpc.ServerOption) (*registerServerWrapper, error) {
	var (
		ln  *Listener
		err error
	)
	s.mutex.Lock()
	defer s.mutex.Unlock()
	sw, ok := s.servers[addr]
	if ok {
		return sw, nil
	}
	if network == networkUnix {
		ln, err = NewUnixListener(addr)
	} else {
		ln, err = NewListener(addr)
	}
	if err != nil {
		log.DefaultLogger.Errorf("create a listener failed: %v", err)
		return nil, err
	}
	srv, err := s.f(conf, options...)
	if err != nil {
		log.DefaultLogger.Errorf("create a registered server failed: %v", err)
		return nil, err
	}
	sw = &registerServerWrapper{
		server: srv,
		ln:     ln,
	}
	s.servers[addr] = sw
	return sw, nil
}

// registerServerWrapper wraps a registered server and its listener
type registerServerWrapper struct {
	once   sync.Once
	server RegisteredServer
	ln     *Listener
}

// Start call the grpc server Serves. If the server is already started, ignore it.
// TODO: support restart and dynamic update, which are not supported at time.
func (rsw *registerServerWrapper) Start(graceful time.Duration) {
	rsw.once.Do(func() {
		go func() {
			if err := rsw.server.Serve(rsw.ln); err != nil {
				log.DefaultLogger.Errorf("grpc server serves error: %v", err)
			}
		}()
		// stop grpc server when mosn process shutdown
		stagemanager.OnGracefulStop(func() error {
			if graceful <= 0 {
				graceful = v2.GrpcDefaultGracefulStopTimeout // use default timeout
			}
			// sync stop grpc server
			timer := time.AfterFunc(graceful, func() {
				rsw.server.Stop()
				log.DefaultLogger.Errorf("[grpc networkFilter] force stop grpc server: %s", rsw.ln.Addr().String())
			})
			defer timer.Stop()
			log.DefaultLogger.Infof("[grpc networkFilter] graceful stopping grpc server: %s", rsw.ln.Addr().String())
			rsw.server.GracefulStop()
			log.DefaultLogger.Infof("[grpc networkFilter] graceful stoped grpc server success:  %s", rsw.ln.Addr().String())
			return nil
		})
	})
}

var stores sync.Map

func RegisterServerHandler(name string, f NewRegisteredServer) bool {
	_, loaded := stores.LoadOrStore(name, &Handler{
		f:       f,
		servers: map[string]*registerServerWrapper{},
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

func ParseGRPC(conf map[string]interface{}) (*v2.GRPC, error) {
	data, _ := json.Marshal(conf)
	v := &v2.GRPC{}
	err := json.Unmarshal(data, v)
	return v, err
}
