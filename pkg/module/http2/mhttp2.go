// Copyright 2018 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package http2

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/http/httpguts"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

var (
	ErrAGAIN = errors.New("EAGAIN")

	//todo: support configuration
	initialConnRecvWindowSize = int32(1 << 30)
)

// Mstream is Http2 Server stream
type MStream struct {
	*stream
	sentContentLen int64
	conn           *MServerConn
	Request        *http.Request
	Response       *http.Response
	Header         http.Header
	Trailers       http.Header
	SendData       buffer.IoBuffer
}

// ID returns stream id
func (ms *MStream) ID() uint32 {
	return ms.id
}

// SendResponse is Http2 Server send response
func (ms *MStream) SendResponse() error {
	ms.conn.closeStream(ms.stream, nil)
	if ms.bodyBytes > 0 {
		ms.conn.sendWindowUpdate(nil, int(ms.bodyBytes))
	}

	isHeadResp := ms.Request.Method == "HEAD"
	var ctype, clen string
	var dataLen int
	if ms.SendData != nil {
		dataLen = ms.SendData.Len()
	}

	rsp := ms.Response

	if clen = rsp.Header.Get("Content-Length"); clen != "" {
		rsp.Header.Del("Content-Length")
		clen64, err := strconv.ParseInt(clen, 10, 64)
		if err == nil && clen64 >= 0 {
			ms.sentContentLen = clen64
		} else {
			clen = ""
		}
	}

	if dataLen == 0 || isHeadResp || !bodyAllowedForStatus(rsp.StatusCode) {
		clen = "0"
	}

	hasContentType := rsp.Header.Get("Content-Type")
	if hasContentType == "" && bodyAllowedForStatus(rsp.StatusCode) && dataLen > 0 {
		ctype = http.DetectContentType(ms.SendData.Bytes())
	}
	var date string
	if ok := rsp.Header.Get("Date"); ok == "" {
		date = time.Now().UTC().Format(http.TimeFormat)
	}

	var trailers []string
	if rsp.Trailer != nil {
		for k, _ := range rsp.Trailer {
			k = http.CanonicalHeaderKey(k)
			if !httpguts.ValidTrailerHeader(k) {
				rsp.Trailer.Del(k)
			} else {
				trailers = append(trailers, k)
			}
		}
	}

	endStream := (dataLen == 0 && len(trailers) == 0) || isHeadResp
	ws := &writeResHeaders{
		streamID:      ms.id,
		httpResCode:   rsp.StatusCode,
		h:             rsp.Header,
		endStream:     endStream,
		contentType:   ctype,
		contentLength: clen,
		date:          date,
	}
	err := ms.conn.writeHeaders(ws)
	if err != nil {
		return err
	}
	if endStream {
		return nil
	}

	if dataLen > 0 {
		err = ms.conn.Framer.writeData(ms.stream.id, len(trailers) == 0, ms.SendData.Bytes())
		if err != nil {
			return err
		}
		buffer.PutIoBuffer(ms.SendData)
	}

	if len(trailers) > 0 {
		ws := &writeResHeaders{
			streamID:  ms.id,
			h:         rsp.Trailer,
			trailers:  trailers,
			endStream: true,
		}
		err = ms.conn.writeHeaders(ws)
	}

	return err
}

func (ms *MStream) Reset() {
	ev := streamError(ms.id, ErrCodeInternal)
	ms.conn.resetStream(ev)
	ms.conn.delStream(ms.id)
}

type MServerConn struct {
	serverConn
	mu sync.Mutex

	Framer *MFramer
	api.Connection
}

// NewserverConn returns a Http2 Server Connection
func NewServerConn(conn api.Connection) *MServerConn {
	sc := new(MServerConn)
	sc.Connection = conn

	// init serverConn
	sc.serverConn.hpackEncoder = hpack.NewEncoder(&sc.headerWriteBuf)
	sc.serverConn.flow.add(initialWindowSize)
	sc.serverConn.inflow.add(initialWindowSize)

	sc.serverConn.advMaxStreams = defaultMaxStreams
	sc.serverConn.streams = make(map[uint32]*stream)
	sc.serverConn.clientMaxStreams = math.MaxUint32
	sc.serverConn.initialStreamSendWindowSize = initialWindowSize
	sc.serverConn.maxFrameSize = initialMaxFrameSize
	sc.serverConn.headerTableSize = initialHeaderTableSize

	sc.serverConn.pushEnabled = false

	// init MFramer
	fr := new(MFramer)
	fr.Framer.ReadMetaHeaders = hpack.NewDecoder(initialHeaderTableSize, nil)
	fr.Framer.MaxHeaderListSize = http.DefaultMaxHeaderBytes
	fr.Framer.SetMaxReadFrameSize(defaultMaxReadFrameSize)
	fr.Connection = conn

	sc.Framer = fr
	return sc
}

// Init send settings frame and window update
func (sc *MServerConn) Init() error {
	settings := writeSettings{
		{SettingMaxFrameSize, defaultMaxReadFrameSize},
		{SettingMaxConcurrentStreams, defaultMaxStreams},
		{SettingMaxHeaderListSize, http.DefaultMaxHeaderBytes},
		{SettingInitialWindowSize, uint32(initialConnRecvWindowSize)},
	}

	err := sc.Framer.writeSettings(settings)
	if err != nil {
		return err
	}
	sc.unackedSettings++

	// Each connection starts with intialWindowSize inflow tokens.
	// If a higher value is configured, we add more tokens.
	if diff := initialConnRecvWindowSize - initialWindowSize; diff > 0 {
		sc.sendWindowUpdate(nil, int(diff))
	}

	return nil
}

func (sc *MServerConn) getStream(id uint32) *stream {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.streams[id]
}

func (sc *MServerConn) delStream(id uint32) bool {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if _, ok := sc.streams[id]; ok {
		delete(sc.streams, id)
		return true
	}
	return false
}

func (sc *MServerConn) setStream(id uint32, ms *stream) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.streams[id] = ms
}

// sendWindowUpdate Http2 server send window update frame
func (sc *MServerConn) sendWindowUpdate(st *stream, n int) {
	// "The legal range for the increment to the flow control
	// window is 1 to 2^31-1 (2,147,483,647) octets."
	// A Go Read call on 64-bit machines could in theory read
	// a larger Read than this. Very unlikely, but we handle it here
	// rather than elsewhere for now.
	const maxUint31 = 1<<31 - 1
	for n >= maxUint31 {
		sc.sendWindowUpdate32(st, maxUint31)
		n -= maxUint31
	}
	sc.sendWindowUpdate32(st, int32(n))
}

func (sc *MServerConn) sendWindowUpdate32(st *stream, n int32) {
	if n == 0 {
		return
	}
	if n < 0 {
		panic("negative update")
	}
	var streamID uint32
	if st != nil {
		streamID = st.id
	}
	sc.Framer.writeWindowUpdate(streamID, uint32(n))
	var ok bool
	if st == nil {
		ok = sc.inflow.add(n)
	} else {
		ok = st.inflow.add(n)
	}
	if !ok {
		panic("internal error; sent too many window updates without decrements?")
	}
}

func (sc *MServerConn) writeHeaders(w *writeResHeaders) error {
	enc, buf := sc.hpackEncoder, &sc.headerWriteBuf
	sc.mu.Lock()
	defer sc.mu.Unlock()

	buf.Reset()
	if w.httpResCode != 0 {
		encKV(enc, ":status", httpCodeString(w.httpResCode))
	}

	encodeHeaders(enc, w.h, w.trailers)

	if w.contentType != "" {
		encKV(enc, "content-type", w.contentType)
	}
	if w.contentLength != "" {
		encKV(enc, "content-length", w.contentLength)
	}
	if w.date != "" {
		encKV(enc, "date", w.date)
	}

	headerBlock := buf.Bytes()
	if len(headerBlock) == 0 && w.trailers == nil {
		panic("unexpected empty hpack")
	}

	const maxFrameSize = 16384
	//const maxFrameSize = 100

	first := true
	var err error
	for len(headerBlock) > 0 {
		frag := headerBlock
		if len(frag) > maxFrameSize {
			frag = frag[:maxFrameSize]
		}
		headerBlock = headerBlock[len(frag):]
		if first {
			err = sc.Framer.writeHeaders(HeadersFrameParam{
				StreamID:      w.streamID,
				BlockFragment: frag,
				EndStream:     w.endStream,
				EndHeaders:    len(headerBlock) == 0,
			})
		} else {
			err = sc.Framer.writeContinuation(w.streamID, len(headerBlock) == 0, frag)
		}
		first = false
		if err != nil {
			return err
		}
	}
	return nil
}

// HandleFrame is Http2 Server handles Frame
func (sc *MServerConn) HandleFrame(ctx context.Context, f Frame) (*MStream, []byte, bool, bool, error) {
	var err error
	var ms *MStream
	var endStream, trailer bool
	var data []byte
	switch f := f.(type) {
	case *SettingsFrame:
		err = sc.processSettings(f)
	case *MetaHeadersFrame:
		ms, trailer, endStream, err = sc.processHeaders(ctx, f)
	case *WindowUpdateFrame:
		err = sc.processWindowUpdate(f)
	case *PingFrame:
		err = sc.processPing(f)
	case *DataFrame:
		data = f.Data()
		endStream, err = sc.processData(ctx, f)
	case *RSTStreamFrame:
		err = sc.processResetStream(f)
		if err == nil {
			err = streamError(f.StreamID, f.ErrCode)
		}
	case *PriorityFrame:
		err = sc.processPriority(f)
	case *GoAwayFrame:
		err = sc.processGoAway(f)
	case *PushPromiseFrame:
		// A client cannot push. Thus, servers MUST treat the receipt of a PUSH_PROMISE
		// frame as a connection error (Section 5.4.1) of type PROTOCOL_ERROR.
		err = http2.ConnectionError(http2.ErrCodeProtocol)
	default:
		err = fmt.Errorf("http2: server ignoring frame: %v", f.Header())
	}

	if err != nil {
		switch ev := err.(type) {
		case StreamError:
			sc.resetStream(ev)
			st := sc.getStream(ev.StreamID)
			sc.closeStream(st, ev.Cause)
		case goAwayFlowError:
			sc.goAway(ErrCodeFlowControl, nil)
		case ConnectionError:
			sc.goAway(ErrCode(ev), nil)
		default:
			sc.goAway(ErrCodeProtocol, nil)
		}
	}

	return ms, data, trailer, endStream, err
}

func (sc *MServerConn) HandleError(ctx context.Context, f Frame, err error) {
	return
}

// processHeaders processes Headers Frame
func (sc *MServerConn) processHeaders(ctx context.Context, f *MetaHeadersFrame) (*MStream, bool, bool, error) {
	id := f.StreamID
	if sc.inGoAway {
		// Ignore.
		return nil, false, false, nil
	}

	if id%2 != 1 {
		return nil, false, false, ConnectionError(ErrCodeProtocol)
	}
	// A HEADERS frame can be used to create a new stream or
	// send a trailer for an open one. If we already have a stream
	// open, let it process its own HEADERS frame (trailers at this
	// point, if it's valid).
	if st := sc.getStream(f.StreamID); st != nil {
		if st.resetQueued {
			// We're sending RST_STREAM to close the stream, so don't bother
			// processing this frame.
			return nil, false, false, nil
		}
		err := st.mprocessTrailerHeaders(ctx, f)
		return nil, true, true, err
	}

	// [...] The identifier of a newly established stream MUST be
	// numerically greater than all streams that the initiating
	// endpoint has opened or reserved. [...]  An endpoint that
	// receives an unexpected stream identifier MUST respond with
	// a connection error (Section 5.4.1) of type PROTOCOL_ERROR.
	if id <= sc.maxClientStreamID {
		return nil, false, false, ConnectionError(ErrCodeProtocol)
	}
	sc.maxClientStreamID = id

	// http://tools.ietf.org/html/rfc7540#section-5.1.2
	// [...] Endpoints MUST NOT exceed the limit set by their peer. An
	// endpoint that receives a HEADERS frame that causes their
	// advertised concurrent stream limit to be exceeded MUST treat
	// this as a stream error (Section 5.4.2) of type PROTOCOL_ERROR
	// or REFUSED_STREAM.
	if sc.curClientStreams+1 > sc.advMaxStreams {
		if sc.unackedSettings == 0 {
			// They should know better.
			return nil, false, false, streamError(id, ErrCodeProtocol)
		}
		// Assume it's a network race, where they just haven't
		// received our last SETTINGS update. But actually
		// this can't happen yet, because we don't yet provide
		// a way for users to adjust server parameters at
		// runtime.
		return nil, false, false, streamError(id, ErrCodeRefusedStream)
	}

	initialState := stateOpen
	if f.StreamEnded() {
		initialState = stateHalfClosedRemote
	}
	st := sc.newStream(id, 0, initialState)

	ms := &MStream{
		stream: st,
		conn:   sc,
	}

	req, err := sc.processRequest(st, f)
	if err != nil {
		return nil, false, false, err
	}

	st.reqTrailer = req.Trailer
	req.Trailer = st.reqTrailer
	if st.reqTrailer != nil {
		st.trailer = make(http.Header)
	}
	st.declBodyBytes = req.ContentLength

	ms.Request = req
	return ms, false, f.StreamEnded(), nil
}

// mprocessTrailerHeaders Processes trailer headers
func (st *stream) mprocessTrailerHeaders(ctx context.Context, f *MetaHeadersFrame) error {
	sc := st.sc
	if st.gotTrailerHeader {
		return ConnectionError(ErrCodeProtocol)
	}
	st.gotTrailerHeader = true
	if !f.StreamEnded() {
		return streamError(st.id, ErrCodeProtocol)
	}

	if len(f.PseudoFields()) > 0 {
		return streamError(st.id, ErrCodeProtocol)
	}
	if st.trailer != nil {
		for _, hf := range f.RegularFields() {
			key := sc.canonicalHeader(hf.Name)
			if !httpguts.ValidTrailerHeader(key) {
				return streamError(st.id, ErrCodeProtocol)
			}
			st.trailer[key] = append(st.trailer[key], hf.Value)
		}
	}
	st.copyTrailersToHandlerRequest()
	st.state = stateHalfClosedRemote
	return nil
}

// processRequest processes headers frame and build http.Request for Http2 Server
func (sc *MServerConn) processRequest(st *stream, f *MetaHeadersFrame) (*http.Request, error) {
	rp := requestParam{
		method:    f.PseudoValue("method"),
		scheme:    f.PseudoValue("scheme"),
		authority: f.PseudoValue("authority"),
		path:      f.PseudoValue("path"),
	}

	isConnect := rp.method == "CONNECT"
	if isConnect {
		if rp.path != "" || rp.scheme != "" || rp.authority == "" {
			return nil, streamError(f.StreamID, ErrCodeProtocol)
		}
	} else if rp.method == "" || rp.path == "" || (rp.scheme != "https" && rp.scheme != "http") {
		// See 8.1.2.6 Malformed Requests and Responses:
		//
		// Malformed requests or responses that are detected
		// MUST be treated as a stream error (Section 5.4.2)
		// of type PROTOCOL_ERROR."
		//
		// 8.1.2.3 Request Pseudo-Header Fields
		// "All HTTP/2 requests MUST include exactly one valid
		// value for the :method, :scheme, and :path
		// pseudo-header fields"
		return nil, streamError(f.StreamID, ErrCodeProtocol)
	}

	bodyOpen := !f.StreamEnded()
	if rp.method == "HEAD" && bodyOpen {
		// HEAD requests can't have bodies
		return nil, streamError(f.StreamID, ErrCodeProtocol)
	}

	rp.header = make(http.Header)
	for _, hf := range f.RegularFields() {
		rp.header.Add(sc.canonicalHeader(hf.Name), hf.Value)
	}
	if rp.authority == "" {
		rp.authority = rp.header.Get("Host")
	}

	needsContinue := rp.header.Get("Expect") == "100-continue"
	if needsContinue {
		rp.header.Del("Expect")
	}
	// Merge Cookie headers into one "; "-delimited value.
	if cookies := rp.header["Cookie"]; len(cookies) > 1 {
		rp.header.Set("Cookie", strings.Join(cookies, "; "))
	}

	// Setup Trailers
	var trailer http.Header
	for _, v := range rp.header["Trailer"] {
		for _, key := range strings.Split(v, ",") {
			key = http.CanonicalHeaderKey(strings.TrimSpace(key))
			switch key {
			case "Transfer-Encoding", "Trailer", "Content-Length":
				// Bogus. (copy of http1 rules)
				// Ignore.
			default:
				if trailer == nil {
					trailer = make(http.Header)
				}
				trailer[key] = nil
			}
		}
	}
	delete(rp.header, "Trailer")

	var url_ *url.URL
	var requestURI string
	if rp.method == "CONNECT" {
		url_ = &url.URL{Host: rp.authority}
		requestURI = rp.authority // mimic HTTP/1 server behavior
	} else {
		var err error
		url_, err = url.ParseRequestURI(rp.path)
		if err != nil {
			return nil, streamError(st.id, ErrCodeProtocol)
		}
		requestURI = rp.path
	}

	req := &http.Request{
		Method:     rp.method,
		URL:        url_,
		RemoteAddr: sc.remoteAddrStr,
		Header:     rp.header,
		RequestURI: requestURI,
		Proto:      "HTTP/2.0",
		ProtoMajor: 2,
		ProtoMinor: 0,
		TLS:        nil,
		Host:       rp.authority,
		Body:       nil,
		Trailer:    trailer,
	}

	if bodyOpen {
		if vv, ok := rp.header["Content-Length"]; ok {
			req.ContentLength, _ = strconv.ParseInt(vv[0], 10, 64)
		} else {
			req.ContentLength = -1
		}
	}

	return req, nil
}

func (sc *MServerConn) newStream(id, pusherID uint32, state streamState) *stream {
	if id == 0 {
		panic("internal error: cannot create stream with id 0")
	}

	st := &stream{
		id:    id,
		state: state,
		sc:    &sc.serverConn,
	}
	st.flow.conn = &sc.flow // link to conn-level counter
	st.flow.add(initialConnRecvWindowSize)
	st.inflow.conn = &sc.inflow // link to conn-level counter
	st.inflow.add(initialConnRecvWindowSize)

	sc.setStream(id, st)

	if st.isPushed() {
		sc.curPushedStreams++
	} else {
		sc.curClientStreams++
	}

	return st
}

func (sc *MServerConn) closeStream(st *stream, err error) {
	if st == nil {
		return
	}
	if sc.delStream(st.id) {
		st.state = stateClosed
		if st.isPushed() {
			sc.curPushedStreams--
		} else {
			sc.curClientStreams--
		}
	}
}

func (sc *MServerConn) state(streamID uint32) (streamState, *stream) {
	// http://tools.ietf.org/html/rfc7540#section-5.1
	if st := sc.getStream(streamID); st != nil {
		return st.state, st
	}
	// "The first use of a new stream identifier implicitly closes all
	// streams in the "idle" state that might have been initiated by
	// that peer with a lower-valued stream identifier. For example, if
	// a client sends a HEADERS frame on stream 7 without ever sending a
	// frame on stream 5, then stream 5 transitions to the "closed"
	// state when the first frame for stream 7 is sent or received."
	if streamID%2 == 1 {
		if streamID <= sc.maxClientStreamID {
			return stateClosed, nil
		}
	} else {
		if streamID <= sc.maxPushPromiseID {
			return stateClosed, nil
		}
	}
	return stateIdle, nil
}

// processData processes Data Frame for Http2 Server
func (sc *MServerConn) processData(ctx context.Context, f *DataFrame) (bool, error) {
	if sc.inGoAway && sc.goAwayCode != ErrCodeNo {
		return false, nil
	}
	data := f.Data()

	// "If a DATA frame is received whose stream is not in "open"
	// or "half closed (local)" state, the recipient MUST respond
	// with a stream error (Section 5.4.2) of type STREAM_CLOSED."
	id := f.Header().StreamID
	state, st := sc.state(id)
	if id == 0 || state == stateIdle {
		// Section 5.1: "Receiving any frame other than HEADERS
		// or PRIORITY on a stream in this state MUST be
		// treated as a connection error (Section 5.4.1) of
		// type PROTOCOL_ERROR."
		return false, ConnectionError(ErrCodeProtocol)
	}
	if st == nil || state != stateOpen || st.gotTrailerHeader || st.resetQueued {
		// This includes sending a RST_STREAM if the stream is
		// in stateHalfClosedLocal (which currently means that
		// the http.Handler returned, so it's done reading &
		// done writing). Try to stop the client from sending
		// more DATA.

		// But still enforce their connection-level flow control,
		// and return any flow control bytes since we're not going
		// to consume them.
		if sc.inflow.available() < int32(f.Length) {
			return false, streamError(id, ErrCodeFlowControl)
		}
		// Deduct the flow control from inflow, since we're
		// going to immediately add it back in
		// sendWindowUpdate, which also schedules sending the
		// frames.
		sc.inflow.take(int32(f.Length))
		sc.sendWindowUpdate(nil, int(f.Length)) // conn-level

		if st != nil && st.resetQueued {
			// Already have a stream error in flight. Don't send another.
			return false, nil
		}
		return false, streamError(id, ErrCodeStreamClosed)
	}

	// Sender sending more than they'd declared?
	if st.declBodyBytes != -1 && st.bodyBytes+int64(len(data)) > st.declBodyBytes {
		return false, streamError(id, ErrCodeStreamClosed)
	}

	if f.Length > 0 {
		// Check whether the client has flow control quota.
		if st.inflow.available() < int32(f.Length) {
			return false, streamError(id, ErrCodeFlowControl)
		}
		st.inflow.take(int32(f.Length))

		// Return any padded flow control now, since we won't
		// refund it later on body reads.
		if pad := int32(f.Length) - int32(len(data)); pad > 0 {
			sc.sendWindowUpdate32(nil, pad)
			sc.sendWindowUpdate32(st, pad)
		}

		st.bodyBytes += int64(len(data))
	}
	if f.StreamEnded() {
		st.state = stateHalfClosedRemote
	}
	return f.StreamEnded(), nil
}

// processSettings processes Settings Frame for Http2 Server
func (sc *MServerConn) processSettings(f *SettingsFrame) error {
	if f.IsAck() {
		sc.unackedSettings--
		if sc.unackedSettings < 0 {
			// Why is the peer ACKing settings we never sent?
			// The spec doesn't mention this case, but
			// hang up on them anyway.
			return ConnectionError(ErrCodeProtocol)
		}
		return nil
	}
	if err := f.ForeachSetting(sc.processSetting); err != nil {
		return err
	}
	buf := buffer.NewIoBuffer(frameHeaderLen)
	sc.Framer.startWrite(buf, FrameSettings, FlagSettingsAck, 0)
	return sc.Framer.endWrite(buf)
}

// processWindowUpdate Processes WindowUpdate Frame for Http2 Server
func (sc *MServerConn) processWindowUpdate(f *WindowUpdateFrame) error {
	switch {
	case f.StreamID != 0: // stream-level flow control
		state, st := sc.state(f.StreamID)
		if state == stateIdle {
			// Section 5.1: "Receiving any frame other than HEADERS
			// or PRIORITY on a stream in this state MUST be
			// treated as a connection error (Section 5.4.1) of
			// type PROTOCOL_ERROR."
			return ConnectionError(ErrCodeProtocol)
		}
		if st == nil {
			// "WINDOW_UPDATE can be sent by a peer that has sent a
			// frame bearing the END_STREAM flag. This means that a
			// receiver could receive a WINDOW_UPDATE frame on a "half
			// closed (remote)" or "closed" stream. A receiver MUST
			// NOT treat this as an error, see Section 5.1."
			return nil
		}
		if !st.flow.add(int32(f.Increment)) {
			return streamError(f.StreamID, ErrCodeFlowControl)
		}
	default: // connection-level flow control
		if !sc.flow.add(int32(f.Increment)) {
			return goAwayFlowError{}
		}
	}
	return nil
}

// processPing Processes Ping Frame for Http2 Server
func (sc *MServerConn) processPing(f *PingFrame) error {
	if f.IsAck() {
		// 6.7 PING: " An endpoint MUST NOT respond to PING frames
		// containing this flag."
		return nil
	}
	if f.StreamID != 0 {
		// "PING frames are not associated with any individual
		// stream. If a PING frame is received with a stream
		// identifier field value other than 0x0, the recipient MUST
		// respond with a connection error (Section 5.4.1) of type
		// PROTOCOL_ERROR."
		return ConnectionError(ErrCodeProtocol)
	}
	if sc.inGoAway && sc.goAwayCode != ErrCodeNo {
		return nil
	}
	buf := buffer.NewIoBuffer(frameHeaderLen + 8)
	sc.Framer.startWrite(buf, FramePing, FlagPingAck, 0)
	sc.Framer.writeBytes(buf, f.Data[:])
	return sc.Framer.endWrite(buf)
}

// processResetStream processes Rst Frame for Http2 Server
func (sc *MServerConn) processResetStream(f *RSTStreamFrame) error {
	state, st := sc.state(f.StreamID)
	if state == stateIdle {
		// 6.4 "RST_STREAM frames MUST NOT be sent for a
		// stream in the "idle" state. If a RST_STREAM frame
		// identifying an idle stream is received, the
		// recipient MUST treat this as a connection error
		// (Section 5.4.1) of type PROTOCOL_ERROR.
		return ConnectionError(ErrCodeProtocol)
	}
	if st != nil {
		sc.closeStream(st, streamError(f.StreamID, f.ErrCode))
	}
	return nil
}

// processPriority processes Priority Frame for Http2 Server
func (sc *MServerConn) processPriority(f *PriorityFrame) error {
	if sc.inGoAway {
		return nil
	}
	return nil
}

// // processGoAway processes GoAway Frame for Http2 Server
func (sc *MServerConn) processGoAway(f *GoAwayFrame) error {
	sc.startGracefulShutdownInternal()
	// http://tools.ietf.org/html/rfc7540#section-6.8
	// We should not create any new streams, which means we should disable push.
	sc.pushEnabled = false

	return nil
}

func (sc *MServerConn) startGracefulShutdownInternal() {
	sc.goAway(ErrCodeNo, nil)
}

func (sc *MServerConn) resetStream(se StreamError) error {
	if st := sc.getStream(se.StreamID); st != nil {
		st.resetQueued = true

		buf := buffer.NewIoBuffer(frameHeaderLen + 8)
		sc.Framer.startWrite(buf, FrameRSTStream, 0, se.StreamID)
		sc.Framer.writeUint32(buf, uint32(se.Code))
		return sc.Framer.endWrite(buf)
	}
	return nil
}

func (sc *MServerConn) goAway(code ErrCode, debugData []byte) {
	if sc.inGoAway {
		return
	}
	sc.inGoAway = true
	sc.goAwayCode = code
	buf := buffer.NewIoBuffer(frameHeaderLen + 32)
	sc.Framer.startWrite(buf, FrameGoAway, 0, 0)
	sc.Framer.writeUint32(buf, sc.maxClientStreamID&(1<<31-1))
	sc.Framer.writeUint32(buf, uint32(code))
	sc.Framer.writeBytes(buf, debugData)
	sc.Framer.endWrite(buf)
}

type MClientConn struct {
	ClientConn

	slock sync.Mutex

	Framer *MFramer
	api.Connection
}

// NewClientConn return Http2 Client conncetion
func NewClientConn(conn api.Connection) *MClientConn {
	cc := new(MClientConn)
	cc.Connection = conn

	cc.ClientConn.streams = make(map[uint32]*clientStream)
	cc.ClientConn.pings = make(map[[8]byte]chan struct{})
	cc.ClientConn.wantSettingsAck = true
	cc.ClientConn.nextStreamID = 1
	cc.ClientConn.maxFrameSize = 16 << 10
	cc.ClientConn.initialWindowSize = 65535
	cc.ClientConn.maxConcurrentStreams = 1000
	cc.ClientConn.peerMaxHeaderListSize = 0xffffffffffffffff
	cc.ClientConn.wantSettingsAck = true
	cc.ClientConn.streams = make(map[uint32]*clientStream)

	cc.flow.add(initialWindowSize)
	cc.inflow.add(initialWindowSize)

	fr := new(MFramer)
	fr.ReadMetaHeaders = hpack.NewDecoder(initialHeaderTableSize, nil)
	fr.MaxHeaderListSize = http.DefaultMaxHeaderBytes
	fr.SetMaxReadFrameSize(defaultMaxReadFrameSize)
	fr.Connection = conn
	cc.Framer = fr

	// henc in response to SETTINGS frames?
	cc.henc = hpack.NewEncoder(&cc.hbuf)

	initialSettings := []Setting{
		{ID: SettingEnablePush, Val: 0},
		{ID: SettingInitialWindowSize, Val: transportDefaultStreamFlow},
	}
	if max := http.DefaultMaxHeaderBytes; max != 0 {
		initialSettings = append(initialSettings, Setting{ID: SettingMaxHeaderListSize, Val: uint32(max)})
	}

	cc.Connection.Write(buffer.NewIoBufferBytes(clientPreface))
	cc.Framer.writeSettings(initialSettings)
	cc.Framer.writeWindowUpdate(0, transportDefaultConnFlow)
	cc.inflow.add(transportDefaultConnFlow + initialWindowSize)

	return cc
}

// WriteHeaders wirtes Headers Frame for Http2 Client
func (cc *MClientConn) WriteHeaders(ctx context.Context, req *http.Request, trailers string, endStream bool) (*clientStream, error) {
	if err := checkConnHeaders(req); err != nil {
		return nil, err
	}

	cs := cc.newStream()
	cs.req = req
	hdrs, err := cc.encodeHeaders(req, false, trailers, req.ContentLength)
	if err != nil {
		delete(cc.streams, cs.ID)
		return nil, err
	}
	err = cc.writeHeaders(cs.ID, endStream, int(cc.maxFrameSize), hdrs)
	if err != nil {
		delete(cc.streams, cs.ID)
		return nil, err
	}
	return cs, nil

}

// MClientStream is Http2 Client Stream
type MClientStream struct {
	*clientStream
	conn     *MClientConn
	Request  *http.Request
	SendData buffer.IoBuffer
}

func NewMClientStream(conn *MClientConn, req *http.Request) *MClientStream {
	return &MClientStream{
		conn:    conn,
		Request: req,
	}
}

// GetID returns stream id
func (cc *MClientStream) GetID() uint32 {
	return cc.ID
}

// RoundTrip sends Request for Http2 Client
func (cc *MClientStream) RoundTrip(ctx context.Context) error {
	trailers, err := commaSeparatedTrailers(cc.Request)
	if err != nil {
		return err
	}

	hasTrailers := trailers != ""
	hasBody := cc.SendData != nil && cc.SendData.Len() != 0

	if hasBody {
		cc.Request.ContentLength = int64(cc.SendData.Len())
	}

	endStream := !hasTrailers && !hasBody

	cc.conn.mu.Lock()

	cs, err := cc.conn.WriteHeaders(ctx, cc.Request, trailers, endStream)
	if err != nil {
		cc.conn.mu.Unlock()
		return err
	}
	cc.clientStream = cs
	cc.conn.mu.Unlock()

	if endStream {
		return nil
	}

	if hasBody {
		err = cc.conn.Framer.writeData(cc.ID, !hasTrailers, cc.SendData.Bytes())
		if err != nil {
			return err
		}
		buffer.PutIoBuffer(cc.SendData)
	}

	if hasTrailers {
		var trls []byte
		cc.conn.mu.Lock()
		trls, err = cc.conn.encodeTrailers(cc.Request)
		if err != nil {
			cc.conn.mu.Unlock()
			return err
		}
		err = cc.conn.writeHeaders(cc.ID, true, int(cc.conn.maxFrameSize), trls)
		cc.conn.mu.Unlock()
	}
	return err
}

func (cc *MClientConn) writeHeaders(streamID uint32, endStream bool, maxFrameSize int, hdrs []byte) error {
	first := true // first frame written (HEADERS is first, then CONTINUATION)

	var err error
	for len(hdrs) > 0 {
		chunk := hdrs
		if len(chunk) > maxFrameSize {
			chunk = chunk[:maxFrameSize]
		}
		hdrs = hdrs[len(chunk):]
		endHeaders := len(hdrs) == 0
		if first {
			err = cc.Framer.writeHeaders(HeadersFrameParam{
				StreamID:      streamID,
				BlockFragment: chunk,
				EndStream:     endStream,
				EndHeaders:    endHeaders,
			})
			first = false
		} else {
			err = cc.Framer.writeContinuation(streamID, endHeaders, chunk)
		}

		if err != nil {
			return err
		}
	}
	return nil
}

// must lock
func (cc *MClientConn) newStream() *clientStream {
	cs := &clientStream{
		cc: &cc.ClientConn,
		ID: cc.nextStreamID,
	}
	cs.flow.add(int32(cc.initialWindowSize))
	cs.flow.setConnFlow(&cc.flow)
	cs.inflow.add(transportDefaultStreamFlow)
	cs.inflow.setConnFlow(&cc.inflow)
	cc.streams[cs.ID] = cs
	cc.nextStreamID += 2
	return cs
}

func (sc *MClientConn) HandleError(ctx context.Context, f Frame, err error) {
	return
}

// HandlerFrame handles Frame for Http2 Client
func (sc *MClientConn) HandleFrame(ctx context.Context, f Frame) (*http.Response, []byte, http.Header, bool, error) {
	var err error
	var data []byte
	var endStream bool
	var trailer http.Header
	var rsp *http.Response

	switch f := f.(type) {
	case *SettingsFrame:
		err = sc.processSettings(f)
	case *MetaHeadersFrame:
		rsp, trailer, endStream, err = sc.processHeaders(ctx, f)
	case *WindowUpdateFrame:
		err = sc.processWindowUpdate(f)
	case *PingFrame:
		err = sc.processPing(f)
	case *DataFrame:
		data = f.Data()
		endStream, err = sc.processData(ctx, f)
	case *RSTStreamFrame:
		err = sc.processResetStream(f)
		if err == nil {
			err = streamError(f.StreamID, f.ErrCode)
		}
	case *GoAwayFrame:
		err = sc.processGoAway(f)
	case *PushPromiseFrame:
		// A client cannot push. Thus, servers MUST treat the receipt of a PUSH_PROMISE
		// frame as a connection error (Section 5.4.1) of type PROTOCOL_ERROR.
		err = ConnectionError(http2.ErrCodeProtocol)
	default:
		err = fmt.Errorf("http2: server ignoring frame: %v", f.Header())
	}

	if err != nil {
		switch ev := err.(type) {
		case StreamError:
			sc.resetStream(ev)
		case goAwayFlowError:
		case ConnectionError:
		default:
		}
	}

	return rsp, data, trailer, endStream, err
}

// processHeaders processes headers Frame for Http2 Client
func (cc *MClientConn) processHeaders(ctx context.Context, f *MetaHeadersFrame) (*http.Response, http.Header, bool, error) {
	cs := cc.streamByID(f.StreamID, f.StreamEnded())
	if cs == nil {
		return nil, nil, false, nil
	}
	if !cs.firstByte {
		cs.firstByte = true
	}
	if !cs.pastHeaders {
		cs.pastHeaders = true
	} else {
		if cs.pastTrailers {
			// Too many HEADERS frames for this stream.
			return nil, nil, false, ConnectionError(ErrCodeProtocol)
		}
		cs.pastTrailers = true
		if !f.StreamEnded() {
			// We expect that any headers for trailers also
			// has END_STREAM.
			return nil, nil, false, ConnectionError(ErrCodeProtocol)
		}
		if len(f.PseudoFields()) > 0 {
			// No pseudo header fields are defined for trailers.
			// TODO: ConnectionError might be overly harsh? Check.
			return nil, nil, false, ConnectionError(ErrCodeProtocol)
		}

		trailer := make(http.Header)
		for _, hf := range f.RegularFields() {
			key := http.CanonicalHeaderKey(hf.Name)
			trailer[key] = append(trailer[key], hf.Value)
		}
		return nil, trailer, true, nil
	}

	res, err := cc.handleResponse(cs, f)
	return res, nil, f.StreamEnded(), err

}

// handleResponse returns http.Response
func (cc *MClientConn) handleResponse(cs *clientStream, f *MetaHeadersFrame) (*http.Response, error) {
	if f.Truncated {
		return nil, errResponseHeaderListSize
	}

	status := f.PseudoValue("status")
	if status == "" {
		return nil, errors.New("malformed response from server: missing status pseudo header")
	}
	statusCode, err := strconv.Atoi(status)
	if err != nil {
		return nil, errors.New("malformed response from server: malformed non-numeric status pseudo header")
	}

	header := make(http.Header)
	res := &http.Response{
		Proto:      "HTTP/2.0",
		ProtoMajor: 2,
		Header:     header,
		StatusCode: statusCode,
		Status:     status + " " + http.StatusText(statusCode),
	}
	for _, hf := range f.RegularFields() {
		key := http.CanonicalHeaderKey(hf.Name)
		if key == "Trailer" {
			t := res.Trailer
			if t == nil {
				t = make(http.Header)
				res.Trailer = t
			}
			foreachHeaderElement(hf.Value, func(v string) {
				t[http.CanonicalHeaderKey(v)] = nil
			})
		} else {
			header[key] = append(header[key], hf.Value)
		}
	}

	streamEnded := f.StreamEnded()
	isHead := cs.req.Method == "HEAD"
	if !streamEnded || isHead {
		res.ContentLength = -1
		if clens := res.Header["Content-Length"]; len(clens) == 1 {
			if clen64, err := strconv.ParseInt(clens[0], 10, 64); err == nil {
				res.ContentLength = clen64
			} else {
				// TODO: care? unlike http/1, it won't mess up our framing, so it's
				// more safe smuggling-wise to ignore.
			}
		} else if len(clens) > 1 {
			// TODO: care? unlike http/1, it won't mess up our framing, so it's
			// more safe smuggling-wise to ignore.
		}
	}

	return res, nil
}

// processData processes Data Frame for Http2 Client
func (cc *MClientConn) processData(ctx context.Context, f *DataFrame) (bool, error) {
	cs := cc.streamByID(f.StreamID, f.StreamEnded())
	if cs == nil {
		cc.mu.Lock()
		neverSent := cc.nextStreamID
		cc.mu.Unlock()
		if f.StreamID >= neverSent {
			// We never asked for this.
			cc.logf("http2: Transport received unsolicited DATA frame; closing connection")
			return false, ConnectionError(ErrCodeProtocol)
		}
		// We probably did ask for this, but canceled. Just ignore it.
		// TODO: be stricter here? only silently ignore things which
		// we canceled, but not things which were closed normally
		// by the peer? Tough without accumulating too much state.

		// But at least return their flow control:
		if f.Length > 0 {
			cc.mu.Lock()
			cc.inflow.add(int32(f.Length))
			cc.mu.Unlock()

			cc.Framer.writeWindowUpdate(0, uint32(f.Length))
		}
		return false, ConnectionError(ErrCodeProtocol)
	}
	if !cs.firstByte {
		cc.logf("protocol error: received DATA before a HEADERS frame")
		return false, StreamError{
			StreamID: f.StreamID,
			Code:     ErrCodeProtocol,
		}
	}
	if f.Length > 0 {
		if cs.req.Method == "HEAD" {
			cc.logf("protocol error: received DATA on a HEAD request")

			return false, StreamError{
				StreamID: f.StreamID,
				Code:     ErrCodeProtocol,
			}
		}
		// Check connection-level flow control.
		cc.mu.Lock()
		if cs.inflow.available() >= int32(f.Length) {
			cs.inflow.take(int32(f.Length))
		} else {
			cc.mu.Unlock()
			return false, ConnectionError(ErrCodeFlowControl)
		}
		// Return any padded flow control now, since we won't
		// refund it later on body reads.
		var refund int
		if pad := int(f.Length) - len(f.Data()); pad > 0 {
			refund += pad
		}
		// Return len(data) now if the stream is already closed,
		// since data will never be read.
		didReset := cs.didReset
		if didReset {
			refund += len(f.Data())
		}
		if refund > 0 {
			cc.inflow.add(int32(refund))
			cc.wmu.Lock()
			cc.Framer.writeWindowUpdate(0, uint32(refund))
			if !didReset {
				cs.inflow.add(int32(refund))
				cc.Framer.writeWindowUpdate(cs.ID, uint32(refund))
			}
			cc.wmu.Unlock()
		}
		cc.mu.Unlock()
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	var connAdd, streamAdd int32
	// Check the conn-level first, before the stream-level.
	if v := cc.inflow.available(); v < transportDefaultConnFlow/2 {
		connAdd = transportDefaultConnFlow - v
		cc.inflow.add(connAdd)
	}

	v := int(cs.inflow.available())
	if v < transportDefaultStreamFlow-transportDefaultStreamMinRefresh {
		streamAdd = int32(transportDefaultStreamFlow - v)
		cs.inflow.add(streamAdd)
	}
	if connAdd != 0 || streamAdd != 0 {
		cc.wmu.Lock()
		defer cc.wmu.Unlock()
		if connAdd != 0 {
			cc.Framer.writeWindowUpdate(0, mustUint31(connAdd))
		}
		if streamAdd != 0 {
			cc.Framer.writeWindowUpdate(cs.ID, mustUint31(streamAdd))
		}
	}

	return f.StreamEnded(), nil
}

// processSettings processes Settings Frame for Http2 Client
func (cc *MClientConn) processSettings(f *SettingsFrame) error {
	cc.mu.Lock()
	defer cc.mu.Unlock()

	if f.IsAck() {
		if cc.wantSettingsAck {
			cc.wantSettingsAck = false
			return nil
		}
		return ConnectionError(ErrCodeProtocol)
	}

	err := f.ForeachSetting(func(s Setting) error {
		switch s.ID {
		case SettingMaxFrameSize:
			cc.maxFrameSize = s.Val
		case SettingMaxConcurrentStreams:
			cc.maxConcurrentStreams = s.Val
		case SettingMaxHeaderListSize:
			cc.peerMaxHeaderListSize = uint64(s.Val)
		case SettingInitialWindowSize:
			// Values above the maximum flow-control
			// window size of 2^31-1 MUST be treated as a
			// connection error (Section 5.4.1) of type
			// FLOW_CONTROL_ERROR.
			if s.Val > math.MaxInt32 {
				return ConnectionError(ErrCodeFlowControl)
			}

			// Adjust flow control of currently-open
			// frames by the difference of the old initial
			// window size and this one.
			delta := int32(s.Val) - int32(cc.initialWindowSize)
			for _, cs := range cc.streams {
				cs.flow.add(delta)
			}
			cc.initialWindowSize = s.Val
		default:
		}
		return nil
	})
	if err != nil {
		return err
	}

	buf := buffer.NewIoBuffer(frameHeaderLen)
	cc.Framer.startWrite(buf, FrameSettings, FlagSettingsAck, 0)
	return cc.Framer.endWrite(buf)
}

// processWindowUpdate processes WindowUpdate Frame for Http2 Client
func (cc *MClientConn) processWindowUpdate(f *WindowUpdateFrame) error {
	cs := cc.streamByID(f.StreamID, false)
	if f.StreamID != 0 && cs == nil {
		return nil
	}

	cc.mu.Lock()
	defer cc.mu.Unlock()

	fl := &cc.flow
	if cs != nil {
		fl = &cs.flow
	}
	if !fl.add(int32(f.Increment)) {
		return ConnectionError(ErrCodeFlowControl)
	}
	return nil
}

// processPing processes Ping Frame for Http2 Client
func (cc *MClientConn) processPing(f *PingFrame) error {
	if f.IsAck() {
		cc.mu.Lock()
		defer cc.mu.Unlock()
		// If ack, notify listener if any
		if c, ok := cc.pings[f.Data]; ok {
			close(c)
			delete(cc.pings, f.Data)
		}
		return nil
	}
	buf := buffer.NewIoBuffer(frameHeaderLen + 8)
	cc.Framer.startWrite(buf, FramePing, FlagPingAck, 0)
	cc.Framer.writeBytes(buf, f.Data[:])
	return cc.Framer.endWrite(buf)
}

// processResetStream processes Rst Frame for Http2 Client
func (cc *MClientConn) processResetStream(f *RSTStreamFrame) error {
	cc.streamByID(f.StreamID, true)
	return nil
}

// processGoAway processes GoAway Frame for Http2 Client
func (cc *MClientConn) processGoAway(f *GoAwayFrame) error {
	return nil
}

func (sc *MClientConn) resetStream(se StreamError) error {
	if st := sc.streamByID(se.StreamID, true); st != nil {

		buf := buffer.NewIoBuffer(frameHeaderLen + 8)
		sc.Framer.startWrite(buf, FrameRSTStream, 0, se.StreamID)
		sc.Framer.writeUint32(buf, uint32(se.Code))
		return sc.Framer.endWrite(buf)
	}
	return nil
}

func (cc *MClientConn) streamByID(id uint32, andRemove bool) *clientStream {
	cc.mu.Lock()
	defer cc.mu.Unlock()
	cs := cc.streams[id]
	if andRemove && cs != nil {
		cc.lastActive = time.Now()
		delete(cc.streams, id)
	}
	return cs
}

type MFramer struct {
	Framer
	api.Connection
}

// WriteSettings wirtes Setting Frame
func (fr *MFramer) writeSettings(settings writeSettings) error {
	buf := buffer.NewIoBuffer(len(settings) * 8)
	fr.startWrite(buf, FrameSettings, 0, 0)
	for _, s := range settings {
		fr.writeUint16(buf, uint16(s.ID))
		fr.writeUint32(buf, s.Val)
	}
	return fr.endWrite(buf)
}

// WriteWindowUpdate wirtes WindwoUpdate Frame
func (fr *MFramer) writeWindowUpdate(streamID, incr uint32) error {
	// "The legal range for the increment to the flow control window is 1 to 2^31-1 (2,147,483,647) octets."
	if incr < 1 || incr > 2147483647 {
		return errors.New("illegal window increment value")
	}
	buf := buffer.NewIoBuffer(20)
	fr.startWrite(buf, FrameWindowUpdate, 0, streamID)
	fr.writeUint32(buf, incr)
	return fr.endWrite(buf)
}

// WriteData writes Data Frame
func (fr *MFramer) writeData(streamID uint32, endStream bool, data []byte) error {
	const maxFrameSize = 16384
	//const maxFrameSize = 100

	var err error
	if data == nil {
		err = fr.sendData(streamID, endStream, nil)
		return err
	}
	for len(data) > 0 {
		frag := data
		if len(frag) > maxFrameSize {
			frag = frag[:maxFrameSize]
		}
		data = data[len(frag):]
		if len(data) > 0 {
			err = fr.sendData(streamID, false, frag)
		} else {
			err = fr.sendData(streamID, endStream, frag)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (fr *MFramer) sendData(streamID uint32, endStream bool, data []byte) error {
	if !validStreamID(streamID) {
		return errStreamID
	}
	var flags Flags
	if endStream {
		flags |= FlagDataEndStream
	}

	buf := buffer.NewIoBuffer(len(data) + frameHeaderLen)
	fr.startWrite(buf, FrameData, flags, streamID)
	buf.Write(data)
	return fr.endWrite(buf)
}

// WriteContinuation writes Continuation Frame
func (fr *MFramer) writeContinuation(streamID uint32, endHeaders bool, headerBlockFragment []byte) error {
	if !validStreamID(streamID) {
		return errStreamID
	}
	var flags Flags
	buf := buffer.NewIoBuffer(len(headerBlockFragment) + frameHeaderLen)
	if endHeaders {
		flags |= FlagContinuationEndHeaders
	}
	fr.startWrite(buf, FrameContinuation, flags, streamID)
	buf.Write(headerBlockFragment)
	return fr.endWrite(buf)
}

// WriteHeaders writes headers Frame
func (fr *MFramer) writeHeaders(p HeadersFrameParam) error {
	if !validStreamID(p.StreamID) {
		return errStreamID
	}
	var flags Flags
	buf := buffer.NewIoBuffer(len(p.BlockFragment) + frameHeaderLen + 8)
	if p.PadLength != 0 {
		flags |= FlagHeadersPadded
	}
	if p.EndStream {
		flags |= FlagHeadersEndStream
	}
	if p.EndHeaders {
		flags |= FlagHeadersEndHeaders
	}
	if !p.Priority.IsZero() {
		flags |= FlagHeadersPriority
	}
	fr.startWrite(buf, FrameHeaders, flags, p.StreamID)
	if p.PadLength != 0 {
		fr.writeByte(buf, p.PadLength)
	}
	if !p.Priority.IsZero() {
		v := p.Priority.StreamDep
		if !validStreamIDOrZero(v) {
			return errDepStreamID
		}
		if p.Priority.Exclusive {
			v |= 1 << 31
		}
		fr.writeUint32(buf, v)
		fr.writeByte(buf, p.Priority.Weight)
	}
	buf.Write(p.BlockFragment)
	buf.Write(padZeros[:p.PadLength])
	return fr.endWrite(buf)
}

func (fr *MFramer) writeByte(b buffer.IoBuffer, v byte)     { b.Write([]byte{v}) }
func (fr *MFramer) writeBytes(b buffer.IoBuffer, v []byte)  { b.Write(v) }
func (fr *MFramer) writeUint16(b buffer.IoBuffer, v uint16) { b.Write([]byte{byte(v >> 8), byte(v)}) }
func (fr *MFramer) writeUint32(b buffer.IoBuffer, v uint32) {
	b.Write([]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func (fr *MFramer) startWrite(buf buffer.IoBuffer, ftype FrameType, flags Flags, streamID uint32) {
	// Write the FrameHeader.
	header := []byte{
		0, // 3 bytes of length, filled in in endWrite
		0,
		0,
		byte(ftype),
		byte(flags),
		byte(streamID >> 24),
		byte(streamID >> 16),
		byte(streamID >> 8),
		byte(streamID),
	}
	// header 是否逃逸？
	buf.Write(header)
}

func (fr *MFramer) endWrite(buf buffer.IoBuffer) error {
	// Now that we know the final size, fill in the FrameHeader in
	// the space previously reserved for it. Abuse append.
	length := buf.Len() - frameHeaderLen
	if length >= (1 << 24) {
		return ErrFrameTooLarge
	}
	header := buf.Bytes()
	header[0] = byte(length >> 16)
	header[1] = byte(length >> 8)
	header[2] = byte(length)
	return fr.Connection.Write(buf)
}

func (fr *MFramer) readFrameHeader(ctx context.Context, data buffer.IoBuffer, off int) (FrameHeader, error) {
	if data.Len() < off+frameHeaderLen {
		return FrameHeader{}, ErrAGAIN
	}
	buf := data.Bytes()[off:]
	return FrameHeader{
		Length:   (uint32(buf[0])<<16 | uint32(buf[1])<<8 | uint32(buf[2])),
		Type:     FrameType(buf[3]),
		Flags:    Flags(buf[4]),
		StreamID: binary.BigEndian.Uint32(buf[5:]) & (1<<31 - 1),
		valid:    true,
	}, nil
}

func (fr *MFramer) readMetaFrame(ctx context.Context, hf *HeadersFrame, data buffer.IoBuffer, off int) (*MetaHeadersFrame, int, error) {
	mh := &MetaHeadersFrame{
		HeadersFrame: hf,
	}

	var hc headersOrContinuation = hf
	frag := make([][]byte, 1)
	msize := 0
	for {
		frag = append(frag, hc.HeaderBlockFragment())

		if hc.HeadersEnded() {
			break
		}
		if f, size, err := fr.ReadFrame(ctx, data, off); err != nil {
			return nil, 0, err
		} else {
			msize += size
			hc = f.(*ContinuationFrame) // guaranteed by checkFrameOrder
		}
	}

	var remainSize = fr.maxHeaderListSize()
	var sawRegular bool

	var invalid error // pseudo header field errors
	hdec := fr.ReadMetaHeaders
	hdec.SetEmitEnabled(true)
	hdec.SetMaxStringLength(fr.maxHeaderStringLen())
	hdec.SetEmitFunc(func(hf hpack.HeaderField) {
		if !httpguts.ValidHeaderFieldValue(hf.Value) {
			invalid = headerFieldValueError(hf.Value)
		}
		isPseudo := strings.HasPrefix(hf.Name, ":")
		if isPseudo {
			if sawRegular {
				invalid = errPseudoAfterRegular
			}
		} else {
			sawRegular = true
			if !validWireHeaderFieldName(hf.Name) {
				invalid = headerFieldNameError(hf.Name)
			}
		}

		if invalid != nil {
			hdec.SetEmitEnabled(false)
			return
		}

		size := hf.Size()
		if size > remainSize {
			hdec.SetEmitEnabled(false)
			mh.Truncated = true
			return
		}
		remainSize -= size

		mh.Fields = append(mh.Fields, hf)
	})
	// Lose reference to MetaHeadersFrame:
	defer hdec.SetEmitFunc(func(hf hpack.HeaderField) {})

	for _, f := range frag {
		if _, err := hdec.Write(f); err != nil {
			return nil, 0, ConnectionError(ErrCodeCompression)
		}
	}

	mh.HeadersFrame.headerFragBuf = nil

	if err := hdec.Close(); err != nil {
		return nil, 0, ConnectionError(ErrCodeCompression)
	}
	if invalid != nil {
		fr.errDetail = invalid
		return nil, 0, StreamError{mh.StreamID, ErrCodeProtocol, invalid}
	}
	if err := mh.checkPseudos(); err != nil {
		fr.errDetail = err
		return nil, 0, StreamError{mh.StreamID, ErrCodeProtocol, err}
	}
	return mh, msize, nil
}

// ReadFrame read Frame
func (fr *MFramer) ReadFrame(ctx context.Context, data buffer.IoBuffer, off int) (Frame, int, error) {
	fr.errDetail = nil
	last := fr.lastFrame
	lastHeader := fr.lastHeaderStream
	fh, err := fr.readFrameHeader(ctx, data, off)
	if err != nil {
		return nil, 0, err
	}
	if fh.Length > fr.maxReadSize {
		return nil, 0, ErrFrameTooLarge
	}

	if int(fh.Length) > data.Len()-(off+frameHeaderLen) {
		return nil, 0, ErrAGAIN
	}

	payload := data.Bytes()[off+frameHeaderLen : off+frameHeaderLen+int(fh.Length)]
	f, err := typeFrameParser(fh.Type)(fr.frameCache, fh, payload)
	if err != nil {
		if ce, ok := err.(connError); ok {
			return nil, 0, fr.connError(ce.Code, ce.Reason)
		}
		return nil, 0, err
	}
	if err := fr.checkFrameOrder(f); err != nil {
		return nil, 0, err
	}
	size := frameHeaderLen + int(fh.Length)
	msize := 0
	if fh.Type == FrameHeaders && fr.ReadMetaHeaders != nil {
		f, msize, err = fr.readMetaFrame(ctx, f.(*HeadersFrame), data, off+size)

		if err != nil {
			fr.lastFrame = last
			fr.lastHeaderStream = lastHeader
			if ce, ok := err.(connError); ok {
				return nil, 0, fr.connError(ce.Code, ce.Reason)
			}
			return nil, 0, err
		}
	}

	if fh.Type != FrameContinuation {
		data.Drain(size + msize)
	}

	return f, size + msize, nil
}

// ReadPreface read the client preface
func (fr *MFramer) ReadPreface(data buffer.IoBuffer) error {
	if data.Len() < len(clientPreface) {
		return ErrAGAIN
	}
	if bytes.Equal(data.Bytes()[0:len(clientPreface)], clientPreface) {
		data.Drain(len(clientPreface))
		return nil
	} else {
		return fmt.Errorf("bogus greeting %q", data.Bytes()[0:len(clientPreface)])
	}
}
