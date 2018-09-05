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
//
// invoke dubbogo.codec & dubbogo.transport to send app req & recv provider rsp

package client

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/registry"
)

// Implements the streamer interface
type rpcStream struct {
	sync.RWMutex
	seq        int64
	closed     chan struct{}
	err        error
	serviceURL registry.ServiceURL
	request    Request
	codec      clientCodec
	context    context.Context
}

func (r *rpcStream) isClosed() bool {
	select {
	case <-r.closed:
		return true
	default:
		return false
	}
}

func (r *rpcStream) Context() context.Context {
	return r.context
}

func (r *rpcStream) Request() Request {
	return r.request
}

// 调用rpcStream.clientCodec.WriteRequest函数
func (r *rpcStream) Send(args interface{}, timeout time.Duration) error {
	r.Lock()

	if r.isClosed() {
		r.err = errShutdown
		r.Unlock()
		return errShutdown
	}

	seq := r.seq
	r.seq++
	r.Unlock()

	req := request{
		Version:       r.request.Version(),
		ServicePath:   strings.TrimPrefix(r.serviceURL.Path, "/"),
		Service:       r.request.ServiceConfig().(*registry.ServiceConfig).Service,
		Seq:           seq,
		ServiceMethod: r.request.Method(),
		Timeout:       timeout,
	}

	if err := r.codec.WriteRequest(&req, args); err != nil {
		r.err = err
		return jerrors.Trace(err)
	}

	return nil
}

func (r *rpcStream) Recv(msg interface{}) error {
	r.Lock()
	defer r.Unlock()

	if r.isClosed() {
		r.err = errShutdown
		return errShutdown
	}

	var rsp response
	if err := r.codec.ReadResponseHeader(&rsp); err != nil {
		if err == io.EOF && !r.isClosed() {
			r.err = io.ErrUnexpectedEOF
			return io.ErrUnexpectedEOF
		}
		log.Warn("msg{%v}, err{%s}", msg, jerrors.ErrorStack(err))
		r.err = err
		return jerrors.Trace(err)
	}

	switch {
	case len(rsp.Error) > 0:
		// We've got an error response. Give this to the request;
		// any subsequent requests will get the ReadResponseBody
		// error if there is one.
		if rsp.Error != lastStreamResponseError {
			r.err = serverError(rsp.Error)
		} else {
			r.err = io.EOF
		}
		if err := r.codec.ReadResponseBody(nil); err != nil {
			r.err = jerrors.Trace(err)
		}

	default:
		if err := r.codec.ReadResponseBody(msg); err != nil {
			r.err = jerrors.Trace(err)
		}
	}

	return r.err
}

func (r *rpcStream) Error() error {
	r.RLock()
	defer r.RUnlock()
	return r.err
}

func (r *rpcStream) Close() error {
	select {
	case <-r.closed:
		return nil
	default:
		close(r.closed)
		return r.codec.Close()
	}
}
