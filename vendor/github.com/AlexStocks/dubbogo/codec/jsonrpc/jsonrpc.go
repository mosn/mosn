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

package jsonrpc

import (
	"io"
)

import (
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
)

type jsonCodec struct {
	mt  codec.MessageType
	rwc io.ReadWriteCloser
	c   *clientCodec
	s   *serverCodec
}

func (j *jsonCodec) Close() error {
	return j.rwc.Close()
}

func (j *jsonCodec) String() string {
	return "jsonrpc2-codec"
}

func (j *jsonCodec) Write(m *codec.Message, b interface{}) error {
	switch m.Type {
	case codec.Request:
		return j.c.Write(m, b)
	case codec.Response:
		return j.s.Write(m, b)
	default:
		return jerrors.Errorf("Unrecognised message type: %v", m.Type)
	}
}

func (j *jsonCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	j.mt = mt

	switch mt {
	case codec.Request:
		return j.s.ReadHeader(m)
	case codec.Response:
		return j.c.ReadHeader(m)
	default:
		return jerrors.Errorf("Unrecognised message type: %v", mt)
	}
	return nil
}

func (j *jsonCodec) ReadBody(b interface{}) error {
	switch j.mt {
	case codec.Request:
		return j.s.ReadBody(b)
	case codec.Response:
		return j.c.ReadBody(b)
	default:
		return jerrors.Errorf("Unrecognised message type: %v", j.mt)
	}
	return nil
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &jsonCodec{
		rwc: rwc,
		c:   newClientCodec(rwc),
		s:   newServerCodec(rwc, nil),
	}
}
