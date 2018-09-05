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

package hessian

import (
	"bufio"
	"io"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
)

type hessianCodec struct {
	mt         codec.MessageType
	rwc        io.ReadWriteCloser
	reader     *bufio.Reader
	rspBodyLen int
}

func (h *hessianCodec) Close() error {
	return h.rwc.Close()
}

func (h *hessianCodec) String() string {
	return "hessian-codec"
}

func (h *hessianCodec) Write(m *codec.Message, a interface{}) error {
	switch m.Type {
	case codec.Heartbeat, codec.Request:
		return jerrors.Trace(packRequest(m, a, h.rwc))
	case codec.Response:
		return nil
	default:
		return jerrors.Errorf("Unrecognised message type: %v", m.Type)
	}

	return nil
}

func (h *hessianCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	h.mt = mt

	switch mt {
	case codec.Request:
		return nil
	case codec.Heartbeat, codec.Response:
		buf, err := h.reader.Peek(HEADER_LENGTH)
		if err != nil { // this is impossible
			return jerrors.Trace(err)
		}
		_, err = h.reader.Discard(HEADER_LENGTH)
		if err != nil { // this is impossible
			return jerrors.Trace(err)
		}

		err = unpackResponseHeaer(buf[:], m)
		if err == codec.ErrJavaException {
			log.Warn("got java exception")
			bufSize := h.reader.Buffered()
			if bufSize > 2 {
				expBuf, expErr := h.reader.Peek(bufSize)
				if expErr == nil {
					log.Warn("java exception:%s", string(expBuf[2:bufSize-1]))
				}
			}
		}
		if err != nil {
			return jerrors.Trace(err)
		}
		h.rspBodyLen = m.BodyLen

		return nil

	default:
		return jerrors.Errorf("Unrecognised message type: %v", mt)
	}

	return nil
}

func (h *hessianCodec) ReadBody(ret interface{}) error {
	switch h.mt {
	case codec.Request:
		return nil

	case codec.Heartbeat, codec.Response:
		// remark on 20180611: the heartbeat return is nil
		//if ret == nil {
		//	return jerrors.Errorf("@ret is nil")
		//}

		buf, err := h.reader.Peek(h.rspBodyLen)
		if err == bufio.ErrBufferFull {
			return codec.ErrBodyNotEnough
		}
		if err != nil {
			return jerrors.Trace(err)
		}
		_, err = h.reader.Discard(h.rspBodyLen)
		if err != nil { // this is impossible
			return jerrors.Trace(err)
		}

		if ret != nil {
			if err = unpackResponseBody(buf, ret); err != nil {
				return jerrors.Trace(err)
			}
		}
	}

	return nil
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &hessianCodec{
		rwc:    rwc,
		reader: bufio.NewReader(rwc),
	}
}
