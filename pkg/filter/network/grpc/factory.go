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
	"sync"

	"mosn.io/api"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/network"
)

func init() {
	api.RegisterNetwork(v2.GRPC_NETWORK_FILTER, CreateGRPCServerFilterFactory)
}

// grpcServerFilterFactory contains a protobuf registered grpc server that provides grpc service
// The registered grpc server needs to be implemented independently by the user, and registered to factory.
// A MOSN can contains multiple registered grpc servers, each grpc server has only one instance.
type grpcServerFilterFactory struct {
	server RegisteredServer
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

// CreateGRPCServerFilterFactory returns a grpc network filter factory.
// The conf contains more than v2.GRPC becasue some config is not ready yet. @nejisama
func CreateGRPCServerFilterFactory(conf map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	cfg, err := ParseGRPC(conf)
	if err != nil {
		log.DefaultLogger.Errorf("invalid grpc server config: %v, error: %v", conf, err)
		return nil, err
	}
	// if OptimizeLocalWrite is true, the connection maybe start a goroutine for connection write,
	// which maybe casues some errors when grpc write. so we do not allow this
	if network.OptimizeLocalWrite {
		return nil, errors.New("grpc does not support local optimize write")
	}
	name := cfg.ServerName
	handler := getRegisterServerHandler(name)
	if handler == nil {
		log.DefaultLogger.Errorf("invalid grpc server found: %s", name)
		return nil, ErrUnknownServer
	}
	// TODO: Close should be called when filter's listener is closed
	// NewListener's conf is expected to removed.
	ln := NewListener(conf)
	// TODO: support dynamic upgrade
	srv, err := handler.Start(ln, cfg.GrpcConfig)
	if err != nil {
		return nil, err
	}
	log.DefaultLogger.Debugf("grpc server filter: create a grpc server name: %s", name)
	return &grpcServerFilterFactory{
		server: srv,
		ln:     ln,
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
type NewRegisteredServer func(json.RawMessage) (RegisteredServer, error)

// Handler is a wrapper to call NewRegisteredServer
type Handler struct {
	once sync.Once
	f    NewRegisteredServer
}

// Start call the grpc server Serves. If the server is already started returns an error
func (s *Handler) Start(ln net.Listener, conf json.RawMessage) (srv RegisteredServer, err error) {
	err = ErrDuplicateStarted
	s.once.Do(func() {
		srv, err = s.f(conf)
		if err != nil {
			log.DefaultLogger.Errorf("start a registered server failed: %v", err)
			return
		}
		go func() {
			// TODO: support restart
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
