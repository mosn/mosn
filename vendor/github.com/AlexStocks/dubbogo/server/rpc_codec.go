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
// receive client req stream and decode them into tm(transport.message),
// assign tm to server.request, call Service.Method to handle request
// and then encode server.response packet into byte stream by
// dubbogo.codec and send them to client by transport

package server

import (
	"bytes"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
	"github.com/AlexStocks/dubbogo/codec/jsonrpc"
	"github.com/AlexStocks/dubbogo/transport"
)

type rpcCodec struct {
	socket transport.Socket
	codec  codec.Codec

	pkg *transport.Package
	buf *readWriteCloser
}

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

var (
	defaultCodecs = map[string]codec.NewCodec{
		"application/json":     jsonrpc.NewCodec,
		"application/json-rpc": jsonrpc.NewCodec,
	}
)

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

func newRPCCodec(req *transport.Package, socket transport.Socket, c codec.NewCodec) serverCodec {
	rwc := &readWriteCloser{
		rbuf: bytes.NewBuffer(req.Body),
		wbuf: bytes.NewBuffer(nil),
	}
	r := &rpcCodec{
		buf:    rwc,
		codec:  c(rwc),
		pkg:    req,
		socket: socket,
	}
	return r
}

func (c *rpcCodec) ReadRequestHeader(r *request, first bool) error {
	m := codec.Message{Header: c.pkg.Header}

	if !first {
		var tp transport.Package
		// transport.Socket.Recv
		if err := c.socket.Recv(&tp); err != nil {
			return err
		}
		c.buf.rbuf.Reset()
		if _, err := c.buf.rbuf.Write(tp.Body); err != nil {
			return err
		}

		m.Header = tp.Header
	}

	err := c.codec.ReadHeader(&m, codec.Request)
	r.Service = m.Target
	r.Method = m.Method
	r.Seq = m.ID
	return err
}

func (c *rpcCodec) ReadRequestBody(b interface{}) error {
	return c.codec.ReadBody(b)
}

func (c *rpcCodec) WriteResponse(r *response, body interface{}, last bool) error {
	c.buf.wbuf.Reset()
	m := &codec.Message{
		Target: r.Service,
		Method: r.Method,
		ID:     r.Seq,
		Error:  r.Error,
		Type:   codec.Response,
		Header: map[string]string{},
	}
	if err := c.codec.Write(m, body); err != nil {
		return err
	}

	m.Header["Content-Type"] = c.pkg.Header["Content-Type"]
	return c.socket.Send(&transport.Package{
		Header: m.Header,
		Body:   c.buf.wbuf.Bytes(),
	})
}

func (c *rpcCodec) Close() error {
	c.buf.Close()
	c.codec.Close()
	// fmt.Println("http transport Close invoked in rpcCodec.Close")
	return c.socket.Close()
}
