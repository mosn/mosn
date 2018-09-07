// Copyright (c) 2015 Asim Aslam.
// Copyright (c) 2016 ~ 2018, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"runtime/debug"
	"strconv"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
	"github.com/AlexStocks/dubbogo/common"
	"github.com/AlexStocks/dubbogo/registry"
	"github.com/AlexStocks/dubbogo/transport"
)

// 完成注册任务
type server struct {
	rpc  []*rpcServer // 处理外部请求,改为数组形式,以监听多个地址
	done chan struct{}
	once sync.Once

	sync.RWMutex
	opts     Options            // codec,transport,registry
	handlers map[string]Handler // interface -> Handler
	wg       sync.WaitGroup
}

func newServer(opts ...Option) Server {
	var (
		num int
	)
	options := newOptions(opts...)
	servers := make([]*rpcServer, len(options.ConfList))
	num = len(options.ConfList)
	for i := 0; i < num; i++ {
		servers[i] = initServer()
	}
	return &server{
		opts:     options,
		rpc:      servers,
		handlers: make(map[string]Handler),
		done:     make(chan struct{}),
	}
}

func (s *server) handlePkg(servo interface{}, sock transport.Socket) {
	var (
		ok          bool
		rpc         *rpcServer
		pkg         transport.Package
		err         error
		timeout     uint64
		contentType string
		codecFunc   codec.NewCodec
		codec       serverCodec
		header      map[string]string
		key         string
		value       string
		ctx         context.Context
	)

	if rpc, ok = servo.(*rpcServer); !ok {
		return
	}

	defer func() { // panic执行之前会保证defer被执行
		if r := recover(); r != nil {
			log.Warn("connection{local:%v, remote:%v} panic error:%#v, debug stack:%s",
				sock.LocalAddr(), sock.RemoteAddr(), r, string(debug.Stack()))
		}

		// close socket
		sock.Close() // 在这里保证了整个逻辑执行完毕后，关闭了连接，回收了socket fd
	}()

	for {
		pkg.Reset()
		// 读取请求包
		if err = sock.Recv(&pkg); err != nil {
			return
		}

		// 下面的所有逻辑都是处理请求包，并回复response
		// we use s Content-Type header to identify the codec needed
		contentType = pkg.Header["Content-Type"]

		// codec of jsonrpc & other type etc
		codecFunc, err = s.newCodec(contentType)
		if err != nil {
			sock.Send(&transport.Package{
				Header: map[string]string{
					"Content-Type": "text/plain",
				},
				Body: []byte(err.Error()),
			})
			return
		}

		// !!!! 雷同于consumer/rpc_client中那个关键的一句，把github.com/AlexStocks/dubbogo/transport & github.com/AlexStocks/dubbogo/codec结合了起来
		// newRPCCodec(*transport.Message, transport.Socket, codec.NewCodec)
		codec = newRPCCodec(&pkg, sock, codecFunc)

		// strip our headers
		header = make(map[string]string)
		for key, value = range pkg.Header {
			header[key] = value
		}
		delete(header, "Content-Type")
		delete(header, "Timeout")

		// ctx = metadata.NewContext(context.Background(), header)
		ctx = context.WithValue(context.Background(), common.DUBBOGO_CTX_KEY, header)
		// we use s Timeout header to set a server deadline
		if len(pkg.Header["Timeout"]) > 0 {
			if timeout, err = strconv.ParseUint(pkg.Header["Timeout"], 10, 64); err == nil {
				ctx, _ = context.WithTimeout(ctx, time.Duration(timeout))
			}
		}

		if err = rpc.serveRequest(ctx, codec, contentType); err != nil {
			log.Info("Unexpected error serving request, closing socket: %v", err)
			return
		}
	}
}

func (s *server) newCodec(contentType string) (codec.NewCodec, error) {
	var (
		ok bool
		cf codec.NewCodec
	)
	if cf, ok = s.opts.Codecs[contentType]; ok {
		return cf, nil
	}
	if cf, ok = defaultCodecs[contentType]; ok {
		return cf, nil
	}
	return nil, jerrors.Errorf("Unsupported Content-Type: %s", contentType)
}

func (s *server) Options() Options {
	var (
		opts Options
	)

	s.RLock()
	opts = s.opts
	s.RUnlock()

	return opts
}

/*
type ProviderServiceConfig struct {
	Protocol string // from ServiceConfig, get field{Path} from ServerConfig by s field
	Service string  // from handler, get field{Protocol, Group, Version} from ServiceConfig by s field
	Group   string
	Version string
	Methods string
	Path    string
}

type ServiceConfig struct {
	Protocol string `required:"true",default:"dubbo"` // codec string, jsonrpc etc
	Service string `required:"true"`
	Group   string
	Version string
}

type ServerConfig struct {
	Protocol string `required:"true",default:"dubbo"` // codec string, jsonrpc etc
	IP       string
	Port     int
}
*/

func (s *server) Handle(h Handler) error {
	var (
		i           int
		j           int
		flag        int
		serviceNum  int
		serverNum   int
		err         error
		config      Options
		serviceConf registry.ProviderServiceConfig
	)
	config = s.Options()

	serviceConf.Service = h.Service()
	serviceConf.Version = h.Version()

	flag = 0
	serviceNum = len(config.ServiceConfList)
	serverNum = len(config.ConfList)
	for i = 0; i < serviceNum; i++ {
		if config.ServiceConfList[i].Service == serviceConf.Service &&
			config.ServiceConfList[i].Version == serviceConf.Version {

			serviceConf.Protocol = config.ServiceConfList[i].Protocol
			serviceConf.Group = config.ServiceConfList[i].Group
			// serviceConf.Version = config.ServiceConfList[i].Version
			for j = 0; j < serverNum; j++ {
				if config.ConfList[j].Protocol == serviceConf.Protocol {
					s.Lock()
					serviceConf.Methods, err = s.rpc[j].register(h)
					s.Unlock()
					if err != nil {
						return err
					}

					serviceConf.Path = config.ConfList[j].Address()
					err = config.Registry.Register(serviceConf)
					if err != nil {
						return err
					}
					flag = 1
				}
			}
		}
	}

	if flag == 0 {
		return jerrors.Errorf("fail to register Handler{service:%s, version:%s}", serviceConf.Service, serviceConf.Version)
	}

	s.Lock()
	s.handlers[h.Service()] = h
	s.Unlock()

	return nil
}

func (s *server) Start() error {
	var (
		i         int
		serverNum int
		err       error
		config    Options
		rpc       *rpcServer
		listener  transport.Listener
	)
	config = s.Options()

	serverNum = len(config.ConfList)
	for i = 0; i < serverNum; i++ {
		listener, err = config.Transport.Listen(config.ConfList[i].Address())
		if err != nil {
			return err
		}
		log.Info("Listening on %s", listener.Addr())

		s.Lock()
		rpc = s.rpc[i]
		rpc.listener = listener
		s.Unlock()

		s.wg.Add(1)
		go func(servo *rpcServer) {
			listener.Accept(func(sock transport.Socket) { s.handlePkg(rpc, sock) })
			s.wg.Done()
		}(rpc)

		s.wg.Add(1)
		go func(servo *rpcServer) { // server done goroutine
			var err error
			<-s.done                     // step1: block to wait for done channel(wait server.Stop step2)
			err = servo.listener.Close() // step2: and then close listener
			if err != nil {
				log.Warn("listener{addr:%s}.Close() = error{%#v}", servo.listener.Addr(), err)
			}
			s.wg.Done()
		}(rpc)
	}

	return nil
}

func (s *server) Stop() {
	s.once.Do(func() {
		close(s.done)
		s.wg.Wait()
		if s.opts.Registry != nil {
			s.opts.Registry.Close()
			s.opts.Registry = nil
		}
	})
}

func (s *server) String() string {
	return "dubbogo-server"
}
