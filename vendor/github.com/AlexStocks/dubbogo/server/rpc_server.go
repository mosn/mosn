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
//
// After accept a tcp socket connection, use server to handle client request.
// register & serverRequest !!

package server

import (
	"context"
	"errors"
	"io"
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/common"
	"github.com/AlexStocks/dubbogo/transport"
)

var (
	lastStreamResponseError = errors.New("EOS")
	// A value sent as a placeholder for the server's response value when the server
	// receives an invalid request. It is never decoded by the client since the Response
	// contains an error when it is used.
	invalidRequest = struct{}{}

	// Precompute the reflect type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()
)

type methodType struct {
	sync.Mutex  // protects counters
	method      reflect.Method
	ArgType     reflect.Type
	ReplyType   reflect.Type
	ContextType reflect.Type
	stream      bool
	numCalls    uint
}

func (m *methodType) NumCalls() (n uint) {
	m.Lock()
	n = m.numCalls
	m.Unlock()
	return n
}

type service struct {
	name string        // name of service
	rcvr reflect.Value // receiver of methods for the service
	typ  reflect.Type  // type of the receiver
	// mainly usded in serverRequest{readRequest{readRequestHeader}->call}
	method map[string]*methodType // registered methods, function name -> reflect.function
}

type rpcRequest struct {
	service     string
	method      string
	contentType string
	request     interface{}
	stream      bool
}

func (r *rpcRequest) ContentType() string {
	return r.contentType
}

func (r *rpcRequest) Service() string {
	return r.service
}

func (r *rpcRequest) Method() string {
	return r.method
}

func (r *rpcRequest) Request() interface{} {
	return r.request
}

func (r *rpcRequest) Stream() bool {
	return r.stream
}

// call过程中arg单独列出
type request struct {
	Service string
	Method  string
	Seq     int64 // sequence number chosen by client
}

type response struct {
	Service string
	Method  string
	Seq     int64  // echoes that of the request
	Error   string // error, if any.
}

// server represents an RPC Server.
type rpcServer struct {
	mu         sync.Mutex          // protects the serviceMap
	serviceMap map[string]*service // service name -> service
	freeReq    chan *request
	freeRsp    chan *response
	listener   transport.Listener
}

const (
	FREE_LIST_SIZE = 4 * 1024
)

func initServer() *rpcServer {
	return &rpcServer{
		serviceMap: make(map[string]*service),
		freeReq:    make(chan *request, FREE_LIST_SIZE),
		freeRsp:    make(chan *response, FREE_LIST_SIZE),
	}
}

// Is this an exported - upper case - name?
// 根据首字母是否大写判定是否暴露给外部用户
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// prepareMethod returns a methodType for the provided method or nil
// in case if the method was unsuitable.
// 如果@method不符合调用条件，则返回nil；否则返回methodType
func prepareMethod(method reflect.Method) *methodType {
	mtype := method.Type
	mname := method.Name
	var replyType, argType, contextType reflect.Type
	var stream bool

	// Method must be exported.
	if method.PkgPath != "" {
		return nil
	}

	switch mtype.NumIn() {
	case 3:
		// assuming streaming
		argType = mtype.In(2)
		contextType = mtype.In(1)
		stream = true
	case 4:
		// method that takes a context
		argType = mtype.In(2)
		replyType = mtype.In(3)
		contextType = mtype.In(1)
	default:
		log.Error("method{%s} of mtype{%v} has wrong number of in parameters{%d}", mname, mtype, mtype.NumIn())
		return nil
	}

	if stream {
		// check stream type
		streamType := reflect.TypeOf((*Streamer)(nil)).Elem()
		if !argType.Implements(streamType) { // 判断arg是否实现了Streamer接口
			log.Error("method{%s} argument does not implement Streamer interface{%v}", mname, argType)
			return nil
		}
	} else {
		// if not stream check the replyType

		// First arg need not be a pointer.
		if !isExportedOrBuiltinType(argType) {
			log.Error("method{%s} argument type not exported{%v}", mname, argType)
			return nil
		}

		if replyType.Kind() != reflect.Ptr {
			log.Error("method{%s} reply type not a pointer{%v}", mname, replyType)
			return nil
		}

		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			log.Error("method{%s} reply type not exported{%v}", mname, replyType)
			return nil
		}
	}

	// Method needs one out.
	if mtype.NumOut() != 1 {
		log.Error("method{%s} has wrong number of out parameters{%d}", mname, mtype.NumOut())
		return nil
	}
	// The return type of the method must be error.
	if returnType := mtype.Out(0); returnType != typeOfError {
		log.Error("method{%s}'s return type{%s} is not error", mname, returnType.String())
		return nil
	}
	return &methodType{method: method, ArgType: argType, ReplyType: replyType, ContextType: contextType, stream: stream}
}

func (server *rpcServer) register(rcvr Handler) (string, error) {
	var (
		num     int
		s       *service
		sname   string
		m       int
		method  reflect.Method
		methods string
		mt      *methodType
	)

	server.mu.Lock()
	defer server.mu.Unlock()
	if server.serviceMap == nil {
		server.serviceMap = make(map[string]*service)
	}
	s = new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname = reflect.Indirect(s.rcvr).Type().Name()
	if sname == "" {
		log.Error("rpc: no service name for type{%s}", s.typ.String())
	}
	if !isExported(sname) {
		s := "rpc Register: type " + sname + " is not exported"
		log.Error(s)
		return "", jerrors.New(s)
	}

	sname = rcvr.Service() //!!serviceMap要根据请求包中的interface来查找Handler，所以此处key使用handler.Interface()返回的值
	if _, present := server.serviceMap[sname]; present {
		return "", jerrors.New("rpc: service already defined: " + sname)
	}
	s.name = sname
	s.method = make(map[string]*methodType)

	// Install the methods
	num = s.typ.NumMethod()
	for m = 0; m < num; m++ {
		method = s.typ.Method(m)
		if mt = prepareMethod(method); mt != nil {
			s.method[method.Name] = mt
			methods += method.Name + ","
		}
	}

	if len(s.method) == 0 {
		s := "rpc Register: type " + sname + " has no exported methods of suitable type"
		log.Error(s)
		return "", jerrors.New(s)
	}
	server.serviceMap[s.name] = s

	return common.TrimSuffix(methods, ","), nil
}

// 调用codec.WriteResponse
func (server *rpcServer) sendResponse(sending *sync.Mutex, req *request, reply interface{}, codec serverCodec, errmsg string, last bool) (err error) {
	resp := server.getResponse()
	// Encode the response header
	resp.Service = req.Service
	resp.Method = req.Method
	if errmsg != "" {
		resp.Error = errmsg
		reply = invalidRequest
	}
	resp.Seq = req.Seq
	sending.Lock()
	err = codec.WriteResponse(resp, reply, last)
	if err != nil {
		log.Error("rpc: writing error response{%v}", err)
	}
	sending.Unlock()
	server.freeRsponse(resp)
	return err
}

func (s *service) call(ctx context.Context, server *rpcServer, sending *sync.Mutex, mtype *methodType, req *request, argv, replyv reflect.Value, codec serverCodec, ct string) {
	var (
		err          error
		errmsg       string
		returnValues []reflect.Value
		function     reflect.Value
		r            *rpcRequest
	)

	mtype.Lock()
	mtype.numCalls++
	mtype.Unlock()

	function = mtype.method.Func
	r = &rpcRequest{
		service:     req.Service,
		method:      req.Method,
		contentType: ct,
	}

	// http之类的短连接请求
	if !mtype.stream {
		r.request = argv.Interface()

		fn := func(ctx context.Context, req Request, rsp interface{}) error {
			returnValues = function.Call([]reflect.Value{s.rcvr, mtype.prepareContext(ctx), reflect.ValueOf(req.Request()), reflect.ValueOf(rsp)})

			// The return value for the method is an error.
			if err := returnValues[0].Interface(); err != nil {
				return err.(error)
			}

			return nil
		}

		errmsg = ""
		err = fn(ctx, r, replyv.Interface()) // 调用相关的函数
		if err != nil {
			errmsg = err.Error()
		}

		server.sendResponse(sending, req, replyv.Interface(), codec, errmsg, true)
		server.freeRequest(req)
		return
	}

	// declare a local error to see if we errored out already
	// keep track of the type, to make sure we return
	// the same one consistently
	var lastError error

	stream := &rpcStream{
		context: ctx,
		codec:   codec,
		request: r,
		seq:     req.Seq,
	}

	// Invoke the method, providing a new value for the reply.
	fn := func(ctx context.Context, req Request, stream interface{}) error {
		returnValues = function.Call([]reflect.Value{s.rcvr, mtype.prepareContext(ctx), reflect.ValueOf(stream)})
		if err := returnValues[0].Interface(); err != nil {
			// the function returned an error, we use that
			return err.(error)
		} else if lastError != nil {
			// we had an error inside sendReply, we use that
			return lastError
		} else {
			// no error, we send the special EOS error
			return lastStreamResponseError
		}
	}

	// client.Stream request
	r.stream = true

	errmsg = ""
	if err = fn(ctx, r, stream); err != nil {
		errmsg = err.Error()
	}

	// this is the last packet, we don't do anything with
	// the error here (well sendStreamResponse will log it
	// already)
	server.sendResponse(sending, req, nil, codec, errmsg, true)
	server.freeRequest(req)
}

func (m *methodType) prepareContext(ctx context.Context) reflect.Value {
	if contextv := reflect.ValueOf(ctx); contextv.IsValid() {
		return contextv
	}
	return reflect.Zero(m.ContextType)
}

func (server *rpcServer) serveRequest(ctx context.Context, codec serverCodec, ct string) error {
	sending := new(sync.Mutex)
	service, mtype, req, argv, replyv, keepReading, err := server.readRequest(codec)
	if err != nil {
		if !keepReading {
			if req != nil {
				server.freeRequest(req)
			}
			return err
		}
		// send a response if we actually managed to read a header.
		if req != nil {
			server.sendResponse(sending, req, invalidRequest, codec, err.Error(), true)
			server.freeRequest(req)
		}
		return err
	}
	service.call(ctx, server, sending, mtype, req, argv, replyv, codec, ct)
	return nil
}

func (server *rpcServer) getRequest() *request {
	var req *request
	// Grab a request if available; allocate if not.
	select {
	case req = <-server.freeReq:
		*req = request{} // 清空结构体内的值
	// Got one; nothing more to do.
	default:
		// None free, so allocate a new one.
		req = new(request)
	}

	return req
}

func (server *rpcServer) freeRequest(req *request) {
	if req != nil {
		// Reuse request there's room.
		select {
		case server.freeReq <- req:
		// Buffer on free list; nothing more to do.
		default:
			// Free list full, just carry on.
		}
	}
}

func (server *rpcServer) getResponse() *response {
	var rsp *response
	// Grab a response if available; allocate if not.
	select {
	case rsp = <-server.freeRsp:
		*rsp = response{} // 清空结构体内的值
	// Got one; nothing more to do.
	default:
		// None free, so allocate a new one.
		rsp = new(response)
	}

	return rsp
}

func (server *rpcServer) freeRsponse(rsp *response) {
	if rsp != nil {
		// Reuse response there's room.
		select {
		case server.freeRsp <- rsp:
		// Buffer on free list; nothing more to do.
		default:
			// Free list full, just carry on.
		}

	}
}

// step1: codec.ReadRequestHeader
// step2: codec.ReadBody
func (server *rpcServer) readRequest(codec serverCodec) (service *service, mtype *methodType, req *request, argv, replyv reflect.Value, keepReading bool, err error) {
	service, mtype, req, keepReading, err = server.readRequestHeader(codec)
	if err != nil {
		if !keepReading {
			return
		}
		// discard body
		codec.ReadRequestBody(nil)
		return
	}
	// is it a streaming request? then we don't read the body
	if mtype.stream { // stream package do not have header/body
		codec.ReadRequestBody(nil)
		return
	}

	// Decode the argument value.
	argIsValue := false // if true, need to indirect before calling.
	if mtype.ArgType.Kind() == reflect.Ptr {
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		argIsValue = true
	}
	// argv guaranteed to be a pointer now.
	if err = codec.ReadRequestBody(argv.Interface()); err != nil {
		return
	}
	if argIsValue {
		argv = argv.Elem()
	}

	if !mtype.stream {
		replyv = reflect.New(mtype.ReplyType.Elem())
	}
	return
}

// @keepRaading 为true，则说明读取header成功
func (server *rpcServer) readRequestHeader(codec serverCodec) (service *service, mtype *methodType, req *request, keepReading bool, err error) {
	// Grab the request header.
	req = server.getRequest() // 从server.freeReq中回去一个request
	err = codec.ReadRequestHeader(req, true)
	if err != nil {
		req = nil
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return
		}
		err = jerrors.New("rpc: server cannot decode request: " + err.Error())
		return
	}

	// We read the header successfully. If we see an error now,
	// we can still recover and move on to the next request.
	keepReading = true
	if req.Service == "" || req.Method == "" {
		err = jerrors.New("rpc: service/method request ill-formed: " + req.Service + "/" + req.Method)
		return
	}
	// Look up the request.
	server.mu.Lock()
	service = server.serviceMap[req.Service]
	server.mu.Unlock()
	if service == nil {
		err = jerrors.New("rpc: can't find service " + req.Service)
		return
	}
	mtype = service.method[req.Method]
	if mtype == nil {
		err = jerrors.New("rpc: can't find method " + req.Method + " of service " + req.Service)
	}
	return
}

// 接口名称可以不一样，但是函数表一样就可以了
type serverCodec interface { // 这个interface的函数列表是dubbogo.codec.Codec interface的子集
	// 函数表中的函数名称可以不一样，但是每个函数的参数列表和返回值列表{列表变量个数个数、每个变量类型、变量顺序}一样就行了
	ReadRequestHeader(*request, bool) error
	ReadRequestBody(interface{}) error
	WriteResponse(*response, interface{}, bool) error

	Close() error
}
