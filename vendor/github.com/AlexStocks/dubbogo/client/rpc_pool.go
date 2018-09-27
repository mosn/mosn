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
	"strings"
	"sync"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/transport"
)

type poolConn struct {
	once *sync.Once
	transport.Client
	created int64 // 为0，则说明没有被创建或者被销毁了
}

func (p *poolConn) Close() error {
	err := jerrors.Errorf("close poolConn{%#v} again", p)
	p.once.Do(func() {
		p.Client.Close()
		p.created = 0
		err = nil
	})
	return err
}

type pool struct {
	size int   // 从line 92可见，size是[]*poolConn数组的size
	ttl  int64 // 从line 61 可见，ttl是每个poolConn的有效期时间. pool对象会在getConn时执行ttl检查

	sync.Mutex
	conns map[string][]*poolConn // 从[]*poolConn 可见key是连接地址，而value是对应这个地址的连接数组
}

func newPool(size int, ttl time.Duration) *pool {
	return &pool{
		size:  size,
		ttl:   int64(ttl.Seconds()),
		conns: make(map[string][]*poolConn),
	}
}

func (p *pool) getConn(protocol, addr string, tr transport.Transport, opts ...transport.DialOption) (*poolConn, error) {
	p.Lock()
	var builder strings.Builder

	builder.WriteString(addr)
	builder.WriteString("@")
	builder.WriteString(protocol)

	key := builder.String()

	conns := p.conns[key]
	now := time.Now().Unix()

	for len(conns) > 0 {
		conn := conns[len(conns)-1]
		conns = conns[:len(conns)-1]
		p.conns[key] = conns

		if d := now - conn.created; d > p.ttl {
			conn.Client.Close()
			continue
		}

		p.Unlock()

		return conn, nil
	}

	p.Unlock()

	// create new conn
	// if @tr is httpTransport, then c is httpTransportClient.
	// if @tr is tcpTransport, then c is tcpTransportClient.
	c, err := tr.Dial(addr, opts...)
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	return &poolConn{&sync.Once{}, c, time.Now().Unix()}, nil
}

func (p *pool) release(protocol, addr string, conn *poolConn, err error) {
	if conn == nil || conn.created == 0 {
		return
	}

	// don't store the conn if it has error
	if err != nil {
		conn.Close() // 须经过(poolConn)Close，以防止多次close transport client
		return
	}

	var builder strings.Builder

	builder.WriteString(addr)
	builder.WriteString("@")
	builder.WriteString(protocol)

	key := builder.String()

	// otherwise put it back for reuse
	p.Lock()
	conns := p.conns[key]
	if len(conns) >= p.size {
		p.Unlock()
		conn.Client.Close()
		return
	}
	p.conns[key] = append(conns, conn)
	p.Unlock()
}
