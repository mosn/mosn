// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tls

import (
	"net"
	"testing"
	"time"
)

func BenchmarkConnOutbufWithPool(b *testing.B) {
	data := make([]byte, 10240)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		c := &Conn{config: testConfig.Clone(), conn: mockConn{}}
		c.writeRecord(recordTypeApplicationData, data)
	}
}

var _ net.Conn = mockConn{}

type mockConn struct{}

func (m mockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m mockConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (m mockConn) Close() error {
	return nil
}

func (m mockConn) LocalAddr() net.Addr {
	return nil
}

func (m mockConn) RemoteAddr() net.Addr {
	return nil
}

func (m mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}
