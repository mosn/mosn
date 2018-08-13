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

package tls

import (
	"net"

	"github.com/alipay/sofa-mosn/pkg/log"
)

type conn struct {
	net.Conn
	peek    [1]byte
	haspeek bool
}

// Peek returns 1 byte from connection, without draining any buffered data.
func (c *conn) Peek() []byte {
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
func (c *conn) Read(b []byte) (int, error) {
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
