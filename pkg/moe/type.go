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

package moe

import (
	"io"

	mosnApi "mosn.io/api"

	"mosn.io/envoy-go-extension/pkg/api"
)

type headerMapImpl struct {
	api.HeaderMap
}

func (h *headerMapImpl) Clone() mosnApi.HeaderMap {
	panic("unsupported yet")
}

type bufferImpl struct {
	api.DataBufferBase
}

func (b *bufferImpl) Read(p []byte) (int, error) {
	panic("implement me")
}

func (b *bufferImpl) ReadOnce(r io.Reader) (int64, error) {
	panic("implement me")
}

func (b *bufferImpl) ReadFrom(r io.Reader) (int64, error) {
	panic("implement me")
}

func (b *bufferImpl) Grow(n int) error {
	panic("implement me")
}

func (b *bufferImpl) WriteTo(w io.Writer) (int64, error) {
	panic("implement me")
}

func (b *bufferImpl) Cap() int {
	panic("implement me")
}

func (b *bufferImpl) Clone() mosnApi.IoBuffer {
	panic("implement me")
}

func (b *bufferImpl) Alloc(i int) {
	panic("implement me")
}

func (b *bufferImpl) Free() {
	panic("implement me")
}

func (b *bufferImpl) Count(i int32) int32 {
	panic("implement me")
}

func (b *bufferImpl) EOF() bool {
	panic("implement me")
}

func (b *bufferImpl) SetEOF(eof bool) {
	panic("implement me")
}

func (b *bufferImpl) CloseWithError(err error) {
	panic("implement me")
}
