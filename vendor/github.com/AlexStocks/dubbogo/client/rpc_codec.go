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
// provide interface for rpc_steam;
// encode app packet into byte stream by codec and send them
// to server by transport, and then receive rsp stream and
// decode them into app package.

package client

import (
	"bytes"
	"errors"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
	"github.com/AlexStocks/dubbogo/transport"
)

const (
	lastStreamResponseError = "EOS"
)

// serverError represents an error that has been returned from
// the remote side of the RPC connection.
type serverError string

func (e serverError) Error() string {
	return string(e)
}

// errShutdown holds the specific error for closing/closed connections
var (
	errShutdown = errors.New("connection is shut down")
)

type rpcCodec struct {
	client transport.Client
	codec  codec.Codec

	pkg *transport.Package
	buf *readWriteCloser
}

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

type clientCodec interface {
	WriteRequest(req *request, args interface{}) error
	ReadResponseHeader(*response) error
	ReadResponseBody(interface{}) error

	Close() error
}

type request struct {
	Version       string
	ServicePath   string
	Service       string
	ServiceMethod string // format: "Service.Method"
	Seq           int64  // sequence number chosen by client
	Timeout       time.Duration
}

type response struct {
	ServiceMethod string // echoes that of the Request
	Seq           int64  // echoes that of the request
	Error         string // error, if any.
}

func (rwc *readWriteCloser) Read(p []byte) (n int, err error) {
	return rwc.rbuf.Read(p)
}

func (rwc *readWriteCloser) Write(p []byte) (n int, err error) {
	return rwc.wbuf.Write(p)
}

func (rwc *readWriteCloser) Close() error {
	rwc.rbuf.Reset()
	rwc.wbuf.Reset()
	return nil
}

func newRPCCodec(req *transport.Package, client transport.Client, c codec.NewCodec) *rpcCodec {
	rwc := &readWriteCloser{
		wbuf: bytes.NewBuffer(nil),
		rbuf: bytes.NewBuffer(nil),
	}

	return &rpcCodec{
		buf:    rwc,
		client: client,
		codec:  c(rwc),
		pkg:    req,
	}
}

func (c *rpcCodec) WriteRequest(req *request, args interface{}) error {
	c.buf.wbuf.Reset()
	m := &codec.Message{
		ID:          req.Seq,
		Version:     req.Version,
		ServicePath: req.ServicePath,
		Target:      req.Service,
		Method:      req.ServiceMethod,
		Timeout:     req.Timeout,
		Type:        codec.Request,
		Header:      map[string]string{},
	}
	// Serialization
	if err := c.codec.Write(m, args); err != nil {
		return jerrors.Trace(err)
	}
	// get binary stream
	c.pkg.Body = c.buf.wbuf.Bytes()
	// tcp 层不使用 transport.Package.Header, codec.Write 调用之后其所有内容已经序列化进 transport.Package.Body
	if c.pkg.Header != nil {
		for k, v := range m.Header {
			c.pkg.Header[k] = v
		}
	}
	return jerrors.Trace(c.client.Send(c.pkg))
}

func (c *rpcCodec) ReadResponseHeader(r *response) error {
	var (
		err error
		p   transport.Package
		cm  codec.Message
	)

	c.buf.rbuf.Reset()

	for {
		p.Reset()
		err = c.client.Recv(&p)
		if err != nil {
			return jerrors.Trace(err)
		}
		c.buf.rbuf.Write(p.Body)
		err = c.codec.ReadHeader(&cm, codec.Response)
		if err != codec.ErrHeaderNotEnough {
			break
		}
	}

	r.ServiceMethod = cm.Method
	r.Seq = cm.ID
	r.Error = cm.Error

	return jerrors.Trace(err)
}

func (c *rpcCodec) ReadResponseBody(b interface{}) error {
	var (
		err error
		p   transport.Package
	)

	for {
		err = c.codec.ReadBody(b)
		if err != codec.ErrBodyNotEnough {
			return jerrors.Trace(err)
		}
		p.Reset()
		err = c.client.Recv(&p)
		if err != nil {
			return jerrors.Trace(err)
		}
		c.buf.rbuf.Write(p.Body)
	}

	return nil
}

func (c *rpcCodec) Close() error {
	c.buf.Close()
	c.codec.Close()
	return c.client.Close()
}
