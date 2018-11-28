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

package client

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
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
	"github.com/AlexStocks/dubbogo/selector"
	"github.com/AlexStocks/dubbogo/transport"
)

const (
	CLEAN_CHANNEL_SIZE = 64
)

//////////////////////////////////////////////
// RPC Client
//////////////////////////////////////////////

type empty struct{}

// thread safe
type rpcClient struct {
	ID   int64
	once sync.Once
	opts Options
	pool *pool

	// gc goroutine
	done chan empty
	wg   sync.WaitGroup
	gcCh chan interface{}
}

func newRPCClient(opt ...Option) Client {
	opts := newOptions(opt...)

	t := time.Now()
	rc := &rpcClient{
		ID:   int64(uint32(t.Second() * t.Nanosecond() * common.Goid())),
		opts: opts,
		pool: newPool(opts.PoolSize, opts.PoolTTL),
		done: make(chan empty),
		gcCh: make(chan interface{}, CLEAN_CHANNEL_SIZE),
	}
	log.Info("client initial ID:%d", rc.ID)
	rc.wg.Add(1)
	go rc.gc()

	return rc
}

// rpcClient garbage collector
func (c *rpcClient) gc() {
	var (
		obj interface{}
	)

	defer c.wg.Done()
LOOP:
	for {
		select {
		case <-c.done:
			log.Info("(rpcClient)gc goroutine exit now ...")
			break LOOP
		case obj = <-c.gcCh:
			switch obj.(type) {
			case *rpcStream:
				obj.(*rpcStream).Close() // stream.Close()->rpcCodec.Close->poolConn.Close->httpTransportClient.Close
			default:
				log.Warn("illegal type of gc obj:%+v", obj)
			}
		}
	}
}

func (r *rpcClient) next(request Request, opts CallOptions) (selector.Next, error) {
	// return remote address
	if nil != opts.Next {
		return opts.Next, nil
	}

	// get next nodes from the selector
	return r.opts.Selector.Select(request.ServiceConfig())
}

// 流程
// 1 创建transport.Package 对象 pkg;
// 2 设置msg.Header;
// 3 创建codec对象;
// 4 从连接池中获取一个连接conn;
// 5 创建stream对象;
// 6 启动一个收发goroutine, 调用stream完成网络收发;
// 7 通过一个error channel等待收发goroutine结束流程。
// rpc client -> rpc stream -> rpc codec -> codec + transport
func (c *rpcClient) call(ctx context.Context, reqID int64, service registry.ServiceURL,
	req Request, rsp interface{}, opts CallOptions) error {

	reqTimeout := opts.RequestTimeout
	if len(service.Query.Get("timeout")) != 0 {
		if timeout, err := strconv.Atoi(service.Query.Get("timeout")); err == nil {
			timeoutDuration := time.Duration(timeout) * time.Millisecond
			if timeoutDuration < reqTimeout {
				reqTimeout = timeoutDuration
			}
		}
	}
	if reqTimeout <= 0 {
		reqTimeout = DefaultRequestTimeout
	}

	// 创建 transport package
	pkg := &transport.Package{}
	if c.opts.CodecType != codec.CODECTYPE_DUBBO {
		pkg.Header = make(map[string]string)
		if md, ok := ctx.Value(common.DUBBOGO_CTX_KEY).(map[string]string); ok {
			for k := range md {
				pkg.Header[k] = md[k]
			}

			// set timeout in nanoseconds
			pkg.Header["Timeout"] = fmt.Sprintf("%d", reqTimeout)
			// set the content type for the request
			pkg.Header["Content-Type"] = req.ContentType()
			// set the accept header
			pkg.Header["Accept"] = req.ContentType()
		}
	}

	var (
		gerr  error
		topts *transport.Options
	)
	conn, err := c.pool.getConn(
		c.opts.CodecType.String(),
		service.Location,
		c.opts.Transport,
		transport.WithTimeout(opts.DialTimeout),
		transport.WithPath(service.Path),
	)
	if err != nil {
		return common.InternalServerError("dubbogo.client", fmt.Sprintf("Error sending request: %v", err))
	}
	topts = c.opts.Transport.Options()
	topts.Addrs = append(topts.Addrs, service.Location)
	topts.Timeout = reqTimeout

	stream := &rpcStream{
		seq:        reqID,
		context:    ctx,
		serviceURL: service,
		request:    req,
		closed:     make(chan struct{}),
		codec:      newRPCCodec(pkg, conn, c.opts.newCodec),
	}

	defer func() {
		// defer execution of release
		if req.Stream() {
			// 只缓存长连接
			log.Debug("store connection:{protocol:%s, location:%s, conn:%#v}, gerr:%#v",
				req.Protocol(), service.Location, conn, gerr)
			c.pool.release(req.Protocol(), service.Location, conn, gerr)
		}

		c.gcCh <- stream
	}()

	ch := make(chan error, 1)
	go func() {
		var (
			err error
		)
		defer func() {
			if panicMsg := recover(); panicMsg != nil {
				if msg, ok := panicMsg.(string); ok {
					ch <- common.InternalServerError("dubbogo.client", strconv.Itoa(int(stream.seq))+" request error, panic msg:"+msg)
				} else {
					ch <- common.InternalServerError("dubbogo.client", "request error")
				}
			}
		}()

		// send request
		// 1 stream的send函数调用rpcStream.clientCodec.WriteRequest函数(从line 119可见clientCodec实际是newRPCCodec);
		// 2 rpcCodec.WriteRequest调用了codec.Write(codec.Message, body)，在给request赋值后，然后又调用了transport.Send函数
		// 3 httpTransportClient根据m{header, body}拼凑http.Request{header, body}，然后再调用http.Request.Write把请求以tcp协议的形式发送出去
		if err = stream.Send(req.Args(), reqTimeout); err != nil {
			ch <- err
			return
		}

		// recv response
		// 1 stream.Recv 调用rpcPlusCodec.ReadResponseHeader & rpcCodec.ReadResponseBody;
		// 2 rpcCodec.ReadResponseHeader 先调用httpTransportClient.read，然后再调用codec.ReadHeader
		// 3 rpcCodec.ReadResponseBody 调用codec.ReadBody
		if err = stream.Recv(rsp); err != nil {
			log.Warn("stream.Recv(ID{%d}, req{%#v}, rsp{%#v}) = err{%t}", reqID, req, rsp, err)
			ch <- err
			return
		}

		// success
		ch <- nil
	}()

	select {
	case err := <-ch:
		gerr = err
		return jerrors.Trace(err)
	case <-ctx.Done():
		gerr = ctx.Err()
		return common.NewError("dubbogo.client", fmt.Sprintf("%v", ctx.Err()), 408)
	}
}

// 流程
// 1 从selector中根据service选择一个provider，具体的来说，就是next函数对象;
// 2 构造call函数;
//   2.1 调用next函数返回provider的serviceurl;
//   2.2 调用rpcClient.call()
// 3 根据重试次数的设定，循环调用call，直到有一次成功或者重试
func (c *rpcClient) Call(ctx context.Context, request Request, response interface{}, opts ...CallOption) error {
	reqID := atomic.AddInt64(&c.ID, 1)
	// make a copy of call opts
	callOpts := c.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// get next nodes selection func from the selector
	next, err := c.next(request, callOpts)
	if err != nil {
		log.Error("selector.Select(request{%#v}) = error{%#v}", request, err)
		if err == selector.ErrNotFound {
			return common.NotFound("dubbogo.client", err.Error())
		}

		return common.InternalServerError("dubbogo.client", err.Error())
	}

	// check if we already have a deadline
	d, ok := ctx.Deadline()
	log.Debug("ctx:%#v, d:%#v, ok:%v", ctx, d, ok)
	if !ok {
		// no deadline so we create a new one
		log.Debug("create timeout context, timeout:%v", callOpts.RequestTimeout)
		ctx, _ = context.WithTimeout(ctx, callOpts.RequestTimeout)
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		opt := WithRequestTimeout(d.Sub(time.Now()))
		log.Debug("WithRequestTimeout:%#v", d.Sub(time.Now()))
		opt(&callOpts)
	}

	select {
	case <-ctx.Done():
		return common.NewError("dubbogo.client", fmt.Sprintf("%v", ctx.Err()), 408)
	default:
	}

	call := func(i int) error {
		// select next node
		serviceURL, err := next(reqID)
		if err != nil {
			log.Error("selector.next(request{%#v}, reqID{%d}) = error{%#v}", request, reqID, err)
			if err == selector.ErrNotFound {
				return common.NotFound("dubbogo.client", err.Error())
			}

			return common.InternalServerError("dubbogo.client", err.Error())
		}

		err = c.call(ctx, reqID, *serviceURL, request, response, callOpts)
		log.Debug("@i{%d}, call(ID{%v}, ctx{%v}, serviceURL{%s}, request{%v}, response{%v}) = err{%v}",
			i, reqID, ctx, serviceURL, request, response, jerrors.ErrorStack(err))
		return jerrors.Trace(err)
	}

	var (
		gerr error
		ch   chan error
	)
	ch = make(chan error, callOpts.Retries)
	for i := 0; i < callOpts.Retries; i++ {
		var index = i
		go func() {
			ch <- call(index)
		}()

		select {
		case <-ctx.Done():
			log.Error("reqID{%d}, @i{%d}, ctx.Done(), ctx.Err:%#v", reqID, i, ctx.Err())
			return common.NewError("dubbogo.client", fmt.Sprintf("%v", ctx.Err()), 408)
		case err := <-ch:
			log.Debug("reqID{%d}, err:%+v", reqID, err)
			if err == nil || len(err.Error()) == 0 {
				return nil
			}
			log.Error("reqID{%d}, @i{%d}, err{%+v}", reqID, i, jerrors.ErrorStack(err))
			gerr = jerrors.Trace(err)
		}
	}

	return gerr
}

func (c *rpcClient) Options() Options {
	return c.opts
}

func (c *rpcClient) NewRequest(group, version, service, method string, args interface{}, reqOpts ...RequestOption) Request {
	codecType := c.opts.CodecType.String()
	if dubbogoClientConfigMap[c.opts.CodecType].transportType == codec.TRANSPORT_TCP {
		reqOpts = append(reqOpts, StreamingRequest())
	}
	return newRPCRequest(group, codecType, version, service, method, args, codec2ContentType[codecType], reqOpts...)
}

func (c *rpcClient) String() string {
	return "dubbogo-client"
}

func (c *rpcClient) Close() {
	close(c.done) // notify gc() to close transport connection
	c.wg.Wait()
	c.once.Do(func() {
		if c.opts.Selector != nil {
			c.opts.Selector.Close()
			c.opts.Selector = nil
		}
		if c.opts.Registry != nil {
			c.opts.Registry.Close()
			c.opts.Registry = nil
		}
	})
}
