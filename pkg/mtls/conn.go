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

package mtls

import (
	gotls "crypto/tls"
	"net"

	"encoding/gob"
	"errors"

	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/mtls/crypto/tls"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// mtls.TLSConn -> tls.Conn -> mtls.Conn

// TLSConn represents a secured connection.
// It implements the net.Conn interface.
type TLSConn struct {
	*tls.Conn
}

// Conn is a generic stream-oriented network connection.
// It implements the net.Conn interface.
type Conn struct {
	net.Conn
	peek    [1]byte
	haspeek bool
}

// Peek returns 1 byte from connection, without draining any buffered data.
func (c *Conn) Peek() []byte {
	b := make([]byte, 1, 1)
	n, err := c.Conn.Read(b)
	if n == 0 {
		log.DefaultLogger.Infof("TLS Peek() error: %v", err)
		return nil
	}
	c.peek[0] = b[0]
	c.haspeek = true
	return b
}

// Read reads data from the connection.
func (c *Conn) Read(b []byte) (int, error) {
	peek := 0
	if c.haspeek {
		c.haspeek = false
		b[0] = c.peek[0]
		if len(b) == 1 {
			return 1, nil
		}
		peek = 1
		b = b[peek:]
	}

	n, err := c.Conn.Read(b)
	return n + peek, err
}

// ConnectionState records basic TLS details about the connection.
func (c *TLSConn) ConnectionState() gotls.ConnectionState {
	return c.Conn.GetConnectionState()
}

// GetRawConn returns network connection.
func (c *TLSConn) GetRawConn() net.Conn {
	if c.Conn == nil {
		return nil
	}
	return c.Conn.GetRawConn()
}

// GetTLSInfo returns TLSInfo
func (c *TLSConn) GetTLSInfo(buf types.IoBuffer) int {
	if c == nil {
		return 0
	}
	info := c.Conn.GetTLSInfo()
	if info == nil {
		log.DefaultLogger.Infof("transferTLS failed, TLS handshake is not completed")
		return 0
	}

	log.DefaultLogger.Infof("transferTLS Info: %+v", info)

	size := buf.Len()

	enc := gob.NewEncoder(buf)
	err := enc.Encode(*info)
	if err != nil {
		return 0
	}

	return buf.Len() - size
}

// SetALPN sets ALPN
func (c *TLSConn) SetALPN(alpn string) {
	c.Conn.SetALPN(alpn)
}

// WriteTo writes data
func (c *TLSConn) WriteTo(v *net.Buffers) (int64, error) {
	buffers := (*[][]byte)(v)
	size := 0
	for _, b := range *buffers {
		size += len(b)
	}

	buf := buffer.GetBytes(size)
	off := 0
	for _, b := range *buffers {
		copy((*buf)[off:], b)
		off += len(b)
	}
	*buffers = (*buffers)[:0]

	off = 0
	for off < size {
		l, err := c.Conn.Write((*buf)[off:])
		if err != nil {
			buffer.PutBytes(buf)
			return int64(off), err
		}
		off += l
	}
	buffer.PutBytes(buf)
	return int64(off), nil
}

// GetTLSConn return TLSConn
func GetTLSConn(c net.Conn, b []byte) (net.Conn, error) {
	var info tls.TransferTLSInfo

	buf := buffer.NewIoBufferBytes(b)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(&info)
	if err != nil {
		return nil, err
	}

	log.DefaultLogger.Infof("transferTLSConn Info: %+v", info)

	conn := tls.TransferTLSConn(c, &info)
	if conn == nil {
		return nil, errors.New("TransferTLSConn error")
	}
	mtlsConn := &TLSConn{
		conn,
	}
	return mtlsConn, nil
}
