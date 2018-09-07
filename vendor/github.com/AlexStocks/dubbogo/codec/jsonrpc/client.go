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
	"encoding/json"
	"io"
	"reflect"
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
)

const (
	MAX_JSONRPC_ID = 0x7FFFFFFF
	VERSION        = "2.0"
)

type clientCodec struct {
	dec *json.Decoder // for reading JSON values
	enc *json.Encoder // for writing JSON values
	c   io.Closer

	// temporary work space
	req  clientRequest
	resp clientResponse

	sync.Mutex
	pending map[int64]string
}

type clientRequest struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int64       `json:"id"`
}

type clientResponse struct {
	Version string           `json:"jsonrpc"`
	ID      int64            `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *Error           `json:"error,omitempty"`
}

func (r *clientResponse) reset() {
	r.Version = ""
	r.ID = 0
	r.Result = nil
	r.Error = nil
}

func newClientCodec(conn io.ReadWriteCloser) *clientCodec {
	return &clientCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		c:       conn,
		pending: make(map[int64]string),
	}
}

func (c *clientCodec) Write(m *codec.Message, param interface{}) error {
	// If return error: it will be returned as is for this call.
	// Allow param to be only Array, Slice, Map or Struct.
	// When param is nil or uninitialized Map or Slice - omit "params".
	if param != nil {
		switch k := reflect.TypeOf(param).Kind(); k {
		case reflect.Map:
			if reflect.TypeOf(param).Key().Kind() == reflect.String {
				if reflect.ValueOf(param).IsNil() {
					param = nil
				}
			}
		case reflect.Slice:
			if reflect.ValueOf(param).IsNil() {
				param = nil
			}
		case reflect.Array, reflect.Struct:
		case reflect.Ptr:
			switch k := reflect.TypeOf(param).Elem().Kind(); k {
			case reflect.Map:
				if reflect.TypeOf(param).Elem().Key().Kind() == reflect.String {
					if reflect.ValueOf(param).Elem().IsNil() {
						param = nil
					}
				}
			case reflect.Slice:
				if reflect.ValueOf(param).Elem().IsNil() {
					param = nil
				}
			case reflect.Array, reflect.Struct:
			default:
				return NewError(errInternal.Code, "unsupported param type: Ptr to "+k.String())
			}
		default:
			return NewError(errInternal.Code, "unsupported param type: "+k.String())
		}
	}

	c.req.Version = VERSION
	c.req.Method = m.Method
	// c.req.Params = b
	c.req.Params = param
	c.req.ID = m.ID & MAX_JSONRPC_ID
	c.Lock()
	// c.pending[m.ID] = m.Method // 此处如果用m.ID会导致error: can not find method of response id 280698512
	c.pending[c.req.ID] = m.Method
	c.Unlock()

	return c.enc.Encode(&c.req)
}

func (c *clientCodec) ReadHeader(m *codec.Message) error {
	c.resp.reset()
	if err := c.dec.Decode(&c.resp); err != nil {
		if err == io.EOF {
			log.Debug("c.dec.Decode(c.resp{%v}) = err{%T-%v}, err == io.EOF", c.resp, err, err)
			return err
		}
		log.Debug("c.dec.Decode(c.resp{%v}) = err{%T-%v}, err != io.EOF", c.resp, err, err)
		return NewError(errInternal.Code, err.Error())
	}

	var ok bool
	c.Lock()
	m.Method, ok = c.pending[c.resp.ID]
	if !ok {
		c.Unlock()
		err := jerrors.Errorf("can not find method of response id %v, response error:%v", c.resp.ID, c.resp.Error)
		log.Debug("clientCodec.ReadHeader(@m{%v}) = error{%v}", m, err)
		return err
	}
	delete(c.pending, c.resp.ID)
	c.Unlock()

	m.Error = ""
	m.ID = c.resp.ID
	if c.resp.Error != nil {
		m.Error = c.resp.Error.Error()
	}

	return nil
}

func (c *clientCodec) ReadBody(x interface{}) error {
	if x == nil || c.resp.Result == nil {
		return nil
	}
	return json.Unmarshal(*c.resp.Result, x)
}

func (c *clientCodec) Close() error {
	return c.c.Close()
}
