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
	"sync"
)

import (
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
)

type serverCodec struct {
	encLock sync.Mutex
	dec     *json.Decoder // for reading JSON values
	enc     *json.Encoder // for writing JSON values
	c       io.Closer
	srv     interface{}

	// temporary work space
	req  serverRequest
	resp serverResponse

	// JSON-RPC clients can use arbitrary json values as request IDs.
	// Package rpc expects uint64 request IDs.
	// We assign uint64 sequence numbers to incoming requests
	// but save the original request ID in the pending map.
	// When rpc responds, we use the sequence number in
	// the response to find the original request ID.
	mutex   sync.Mutex
	seq     int64
	pending map[int64]*json.RawMessage
}

// serverRequest represents a JSON-RPC request received by the server.
type serverRequest struct {
	// JSON-RPC protocol.
	Version string `json:"jsonrpc"`
	// A String containing the name of the method to be invoked.
	Method string `json:"method"`
	// A Structured value to pass as arguments to the method.
	Params *json.RawMessage `json:"params"`
	// The request id. MUST be a string, number or null.
	ID *json.RawMessage `json:"id"`
}

// serverResponse represents a JSON-RPC response returned by the server.
type serverResponse struct {
	// JSON-RPC protocol.
	Version string `json:"jsonrpc"`
	// This must be the same id as the request it is responding to.
	ID *json.RawMessage `json:"id"`
	// The Object that was returned by the invoked method. This must be null
	// in case there was an error invoking the method.
	// As per spec the member will be omitted if there was an error.
	Result interface{} `json:"result,omitempty"`
	// An Error object if there was an error invoking the method. It must be
	// null if there was no error.
	// As per spec the member will be omitted if there was no error.
	Error interface{} `json:"error,omitempty"`
}

var (
	null    = json.RawMessage([]byte("null"))
	Version = "2.0"
	// BatchMethod = "JSONRPC2.Batch"
)

// NewServerCodec returns a new rpc.ServerCodec using JSON-RPC 2.0 on conn,
// which will use srv to execute batch requests.
//
// If srv is nil then rpc.DefaultServer will be used.
//
// For most use cases NewServerCodec is too low-level and you should use
// ServeConn instead. You'll need NewServerCodec if you wanna register
// your own object of type named "JSONRPC2" (same as used internally to
// process batch requests) or you wanna use custom rpc server object
// instead of rpc.DefaultServer to process requests on conn.
func newServerCodec(conn io.ReadWriteCloser, srv interface{}) *serverCodec {
	return &serverCodec{
		dec:     json.NewDecoder(conn),
		enc:     json.NewEncoder(conn),
		c:       conn,
		srv:     srv,
		pending: make(map[int64]*json.RawMessage),
	}
}

func (r *serverRequest) reset() {
	r.Version = ""
	r.Method = ""
	if r.Params != nil {
		*r.Params = (*r.Params)[:0]
	}
	if r.ID != nil {
		*r.ID = (*r.ID)[:0]
	}
}

func (r *serverRequest) UnmarshalJSON(raw []byte) error {
	r.reset()
	type req *serverRequest
	if err := json.Unmarshal(raw, req(r)); err != nil {
		return jerrors.New("bad request")
	}

	var o = make(map[string]*json.RawMessage)
	if err := json.Unmarshal(raw, &o); err != nil {
		return jerrors.New("bad request")
	}
	if o["jsonrpc"] == nil || o["method"] == nil {
		return jerrors.New("bad request")
	}
	_, okID := o["id"]
	_, okParams := o["params"]
	if len(o) == 3 && !(okID || okParams) || len(o) == 4 && !(okID && okParams) || len(o) > 4 {
		return jerrors.New("bad request")
	}
	if r.Version != Version {
		return jerrors.New("bad request")
	}
	if okParams {
		if r.Params == nil || len(*r.Params) == 0 {
			return jerrors.New("bad request")
		}
		switch []byte(*r.Params)[0] {
		case '[', '{':
		default:
			return jerrors.New("bad request")
		}
	}
	if okID && r.ID == nil {
		r.ID = &null
	}
	if okID {
		if len(*r.ID) == 0 {
			return jerrors.New("bad request")
		}
		switch []byte(*r.ID)[0] {
		case 't', 'f', '{', '[':
			return jerrors.New("bad request")
		}
	}

	return nil
}

func (c *serverCodec) ReadHeader(m *codec.Message) error {
	// If return error:
	// - codec will be closed
	// So, try to send error reply to client before returning error.
	var raw json.RawMessage
	c.req.reset()
	if err := c.dec.Decode(&raw); err != nil {
		c.encLock.Lock()
		c.enc.Encode(serverResponse{Version: Version, ID: &null, Error: errParse})
		c.encLock.Unlock()
		return err
	}
	// if len(raw) > 0 && raw[0] == '[' {
	// 	c.req.Version = Version
	// 	c.req.Method = BatchMethod
	// 	c.req.Params = &raw
	// 	c.req.ID = &null
	// } else
	if err := json.Unmarshal(raw, &c.req); err != nil {
		if err.Error() == "bad request" {
			c.encLock.Lock()
			c.enc.Encode(serverResponse{Version: Version, ID: &null, Error: errRequest})
			c.encLock.Unlock()
		}
		return err
	}

	m.Method = c.req.Method
	m.Target = m.Header["Path"] // get Path in http_transport.go:Recv:line 242
	if m.Header["HttpMethod"] != "POST" {
		return errMethod
	}

	// JSON request id can be any JSON value;
	// RPC package expects uint64.  Translate to
	// internal uint64 and save JSON on the side.
	c.mutex.Lock()
	c.seq++
	c.pending[c.seq] = c.req.ID
	c.req.ID = nil
	m.ID = c.seq
	c.mutex.Unlock()

	return nil
}

// ReadRequest fills the request object for the RPC method.
//
// ReadRequest parses request parameters in two supported forms in
// accordance with http://www.jsonrpc.org/specification#parameter_structures
//
// by-position: params MUST be an Array, containing the
// values in the Server expected order.
//
// by-name: params MUST be an Object, with member names
// that match the Server expected parameter names. The
// absence of expected names MAY result in an error being
// generated. The names MUST match exactly, including
// case, to the method's expected parameters.
func (c *serverCodec) ReadBody(x interface{}) error {
	// If x!=nil and return error e:
	// - Write() will be called with e.Error() in r.Error
	if x == nil {
		return nil
	}
	if c.req.Params == nil {
		return nil
	}

	var (
		params []byte
		err    error
	)

	// 在这里把请求参数json 字符串转换成了相应的struct
	params = []byte(*c.req.Params)
	// if c.req.Method == BatchMethod {
	// 	return jerrors.Errorf("batch request is not allowed")
	// 	/*
	// 		arg := x.(*BatchArg)
	// 		arg.srv = c.srv
	// 		if err := json.Unmarshal(*c.req.Params, &arg.reqs); err != nil {
	// 			return NewError(errParams.Code, err.Error())
	// 		}
	// 		if len(arg.reqs) == 0 {
	// 			return errRequest
	// 		}
	// 	*/
	// } else
	if err = json.Unmarshal(*c.req.Params, x); err != nil {
		// Note: if c.request.Params is nil it's not an error, it's an optional member.
		// JSON params structured object. Unmarshal to the args object.

		if 2 < len(params) && params[0] == '[' && params[len(params)-1] == ']' {
			// Clearly JSON params is not a structured object,
			// fallback and attempt an unmarshal with JSON params as
			// array value and RPC params is struct. Unmarshal into
			// array containing the request struct.
			params := [1]interface{}{x}
			if err = json.Unmarshal(*c.req.Params, &params); err != nil {
				return NewError(errParams.Code, err.Error())
			}
		} else {
			return NewError(errParams.Code, err.Error())
		}
	}

	return nil
}

func (c *serverCodec) Write(m *codec.Message, x interface{}) error {
	// If return error: nothing happens.
	// In r.Error will be "" or .Error() of error returned by:
	// - ReadBody()
	// - called RPC method
	var (
		err error
	)

	c.mutex.Lock()
	b, ok := c.pending[m.ID]
	if !ok {
		c.mutex.Unlock()
		return jerrors.New("invalid sequence number in response")
	}
	delete(c.pending, m.ID)
	c.mutex.Unlock()

	if b == nil {
		// Notification. Do not respond.
		return nil
	}

	var resp serverResponse = serverResponse{Version: Version, ID: b, Result: x}
	if m.Error == "" {
		if x == nil {
			resp.Result = &null
		} else {
			resp.Result = x
		}
	} else if m.Error[0] == '{' && m.Error[len(m.Error)-1] == '}' {
		// Well& this check for '{'&'}' isn't too strict, but I
		// suppose we're trusting our own RPC methods (this way they
		// can force sending wrong reply or many replies instead
		// of one) and normal errors won't be formatted this way.
		raw := json.RawMessage(m.Error)
		resp.Error = &raw
	} else {
		raw := json.RawMessage(newError(m.Error).Error())
		resp.Error = &raw
	}
	c.encLock.Lock()
	err = c.enc.Encode(resp) // 把resp通过enc关联的conn发送给client
	c.encLock.Unlock()

	return err
}

func (c *serverCodec) Close() error {
	return c.c.Close()
}

/*
tcp stream example:
23:30:30.256489 IP 192.168.35.1.59003 > 192.168.35.3.38081: Flags [P.], seq 1:306, ack 1, win 256, length 305
0x0000:  4500 0159 19bf 4000 4006 588b c0a8 2301  E..Y..@.@.X...#.
0x0010:  c0a8 2303 e67b 94c1 bb8d 1d19 59c2 40ef  ..#..{......Y.@.
0x0020:  5018 0100 05e6 0000 504f 5354 202f 636f  P.......POST./co
0x0030:  6d2e 6f66 7061 792e 6465 6d6f 2e61 7069  m.ofpay.demo.api
0x0040:  2e55 7365 7250 726f 7669 6465 7232 2048  .UserProvider2.H
0x0050:  5454 502f 312e 310d 0a48 6f73 743a 2031  TTP/1.1..Host:.1
0x0060:  3932 2e31 3638 2e33 352e 333a 3338 3038  92.168.35.3:3808
0x0070:  310d 0a55 7365 722d 4167 656e 743a 2047  1..User-Agent:.G
0x0080:  6f2d 6874 7470 2d63 6c69 656e 742f 312e  o-http-client/1.
0x0090:  310d 0a43 6f6e 7465 6e74 2d4c 656e 6774  1..Content-Lengt
0x00a0:  683a 2036 320d 0a41 6363 6570 743a 2061  h:.62..Accept:.a
0x00b0:  7070 6c69 6361 7469 6f6e 2f6a 736f 6e0d  pplication/json.
0x00c0:  0a43 6f6e 7465 6e74 2d54 7970 653a 2061  .Content-Type:.a
0x00d0:  7070 6c69 6361 7469 6f6e 2f6a 736f 6e0d  pplication/json.
0x00e0:  0a54 696d 656f 7574 3a20 3235 3030 3030  .Timeout:.250000
0x00f0:  3030 300d 0a58 2d46 726f 6d2d 4964 3a20  000..X-From-ID:.
0x0100:  7363 7269 7074 0d0a 582d 5573 6572 2d49  script..X-User-I
0x0110:  643a 206a 6f68 6e0d 0a0d 0a7b 226a 736f  d:.john....{"jso
0x0120:  6e72 7063 223a 2232 2e30 222c 226d 6574  nrpc":"2.0","met
0x0130:  686f 6422 3a22 6765 7455 7365 7222 2c22  hod":"getUser","
0x0140:  7061 7261 6d73 223a 5b22 4130 3033 225d  params":["A003"]
0x0150:  2c22 6964 223a 307d 0a                   ,"id":0}.

23:30:30.259835 IP 192.168.35.3.38081 > 192.168.35.1.59003: Flags [P.], seq 1:179, ack 306, win 123, length 178
0x0000:  4500 00da b8a0 4000 4006 ba28 c0a8 2303  E.....@.@..(..#.
0x0010:  c0a8 2301 94c1 e67b 59c2 40ef bb8d 1e4a  ..#....{Y.@....J
0x0020:  5018 007b c821 0000 4854 5450 2f31 2e31  P..{.!..HTTP/1.1
0x0030:  2032 3030 204f 4b0d 0a54 7261 6e73 6665  .200.OK..Transfe
0x0040:  722d 456e 636f 6469 6e67 3a20 6368 756e  r-Encoding:.chun
0x0050:  6b65 640d 0a53 6572 7665 723a 204a 6574  ked..Server:.Jet
0x0060:  7479 2836 2e31 2e32 3629 0d0a 0d0a 3636  ty(6.1.26)....66
0x0070:  0d0a 7b22 6a73 6f6e 7270 6322 3a22 322e  ..{"jsonrpc":"2.
0x0080:  3022 2c22 6964 223a 302c 2272 6573 756c  0","id":0,"resul
0x0090:  7422 3a7b 2269 6422 3a22 4130 3033 222c  t":{"id":"A003",
0x00a0:  226e 616d 6522 3a22 4a6f 6522 2c22 6167  "name":"Joe","ag
0x00b0:  6522 3a34 382c 2274 696d 6522 3a31 3436  e":48,"time":146
0x00c0:  3930 3731 3833 3032 3538 2c22 7365 7822  9071830258,"sex"
0x00d0:  3a22 4d41 4e22 7d7d 0d0a                 :"MAN"}}..
*/
