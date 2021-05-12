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

package common

type IoBuffer interface {
	// Len returns the number of bytes of the unread portion of the buffer;
	// b.Len() == len(b.Bytes()).
	Len() int

	// Bytes returns all bytes from buffer, without draining any buffered data.
	// It can be used to get fixed-length content, such as headers, body.
	// Note: do not change content in return bytes, use write instead
	Bytes() []byte

	// Write appends the contents of p to the buffer, growing the buffer as
	// needed. The return value n is the length of p; err is always nil. If the
	// buffer becomes too large, Write will panic with ErrTooLarge.
	Write(p []byte) (n int, err error)

	// Drain drains a offset length of bytes in buffer.
	// It can be used with Bytes(), after consuming a fixed-length of data
	Drain(offset int)
}

// CommonBuffer is a simple implementation of IoBuffer.
type CommonBuffer struct {
	buf []byte
}

func (c *CommonBuffer) Len() int {
	return len(c.buf)
}

func (c *CommonBuffer) Bytes() []byte {
	return c.buf
}

func (c *CommonBuffer) Write(p []byte) (n int, err error) {
	c.buf = append(c.buf, p...)
	return len(p), nil
}

func (c *CommonBuffer) Drain(offset int) {
	if offset > len(c.buf) {
		return
	}
	c.buf = c.buf[offset:]
}

func NewIoBufferBytes(data []byte) IoBuffer {
	return &CommonBuffer{buf: data}
}
