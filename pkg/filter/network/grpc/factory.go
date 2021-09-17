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

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
	"mosn.io/mosn/pkg/streamfilter"
	"mosn.io/mosn/pkg/types"
	"mosn.io/mosn/pkg/variable"
	"mosn.io/pkg/header"
)

func init() {
	api.RegisterNetwork(v2.GRPC_NETWORK_FILTER, CreateGRPCServerFilterFactory)
}

// grpcServerFilterFactory contains a protobuf registered grpc server that provides grpc service
// The registered grpc server needs to be implemented independently by the user, and registered to factory.
// A MOSN can contains multiple registered grpc servers, each grpc server has only one instance.
type grpcServerFilterFactory struct {
	server              RegisteredServer
	config              *v2.GRPC
	handler             *Handler
	streamFilterFactory streamfilter.StreamFilterFactory
	ln                  *Listener
}

var _ api.NetworkFilterChainFactory = (*grpcServerFilterFactory)(nil)
var _ api.FactoryInitializer = (*grpcServerFilterFactory)(nil)

func (f *grpcServerFilterFactory) CreateFilterChain(ctx context.Context, callbacks api.NetWorkFilterChainFactoryCallbacks) {
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("create a new grpc filter")
	}
	rf := NewGrpcFilter(ctx, f.ln)
	callbacks.AddReadFilter(rf)
}

func (f *grpcServerFilterFactory) Init(param interface{}) error {
	cfg, ok := param.(*v2.Listener)
	if !ok {
		return ErrInvalidConfig
	}
	addr := cfg.AddrConfig
	if addr == "" {
		addr = cfg.Addr.String()
	}
	ln, err := NewListener(addr)
	if err != nil {
		return err
	}
	// GetStreamFilters from listener name
	f.streamFilterFactory = streamfilter.GetStreamFilterManager().GetStreamFilterFactory(cfg.Name)

	//
	opts := []grpc.ServerOption{
		grpc.UnaryInterceptor(f.UnaryInterceptorFilter),
		grpc.StreamInterceptor(f.StreamInterceptorFilter),
	}
	srv, err := f.handler.Start(ln, f.config.GrpcConfig, opts...)
	if err != nil {
		return err
	}
	log.DefaultLogger.Debugf("grpc server filter initialized success")
	// we keep the server but it is not be used currently.
	// maybe it will be used when server stop/restart are supported
	f.server = srv
	f.ln = ln
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

	ctx = mosnctx.WithValue(ctx, types.ContextKeyDownStreamProtocol, api.ProtocolName(grpcName))
	ctx = mosnctx.WithValue(ctx, types.ContextKeyDownStreamHeaders, requestHeader)
	ctx = variable.NewVariableContext(ctx)

	variable.SetVariableValue(ctx, VarGrpcServiceName, info.FullMethod)

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

	responseTrailer := header.CommonHeader{}
	for k, v := range wr.trailer {
		responseTrailer.Set(k, v[0])
	}

	variable.SetVariableValue(ctx, VarGrpcRequestResult, "true")
	if err != nil {
		variable.SetVariableValue(ctx, VarGrpcRequestResult, "false")
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
	ErrUnknownServer    = errors.New("cannot found the grpc server")
	ErrDuplicateStarted = errors.New("grpc server has already started")
	ErrInvalidConfig    = errors.New("invalid config for grpc listener")
)

// CreateGRPCServerFilterFactory returns a grpc network filter factory.
// The conf contains more than v2.GRPC becasue some config is not ready yet. @nejisama
func CreateGRPCServerFilterFactory(conf map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	cfg, err := ParseGRPC(conf)
	if err != nil {
		log.DefaultLogger.Errorf("invalid grpc server config: %v, error: %v", conf, err)
		return nil, err
	}
	// if OptimizeLocalWrite is true, the connection maybe start a goroutine for connection write,
	// which maybe cause some errors when grpc write. so we do not allow this
	if network.OptimizeLocalWrite {
		return nil, errors.New("grpc does not support local optimize write")
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
	Stop()
}

// NewRegisteredServer returns a grpc server that has completed protobuf registration.
// The grpc server Serve function should be called in MOSN.
// NOTICE: some grpc options is not supported, for example, secure options, which are managed by MOSN too.
type NewRegisteredServer func(config json.RawMessage, options ...grpc.ServerOption) (RegisteredServer, error)

// Handler is a wrapper to call NewRegisteredServer
type Handler struct {
	once sync.Once
	f    NewRegisteredServer
}

// Start call the grpc server Serves. If the server is already started returns an error
func (s *Handler) Start(ln net.Listener, conf json.RawMessage, options ...grpc.ServerOption) (srv RegisteredServer, err error) {
	err = ErrDuplicateStarted
	s.once.Do(func() {
		srv, err = s.f(conf, options...)
		if err != nil {
			log.DefaultLogger.Errorf("start a registered server failed: %v", err)
			return
		}
		go func() {
			// TODO: support restart and dynamic update, which are not supported at time.
			if r := srv.Serve(ln); r != nil {
				log.DefaultLogger.Errorf("grpc server serves error: %v", r)
			}
		}()
		err = nil
	})
	return srv, err
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

func ParseGRPC(conf map[string]interface{}) (*v2.GRPC, error) {
	data, _ := json.Marshal(conf)
	v := &v2.GRPC{}
	err := json.Unmarshal(data, v)
	return v, err
}
