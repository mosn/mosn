package http2

import (
	"bytes"
	"fmt"
	"golang.org/x/net/http/httpguts"
	"io"
	"math"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/module/http2/hpack"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"net/http"
	"strconv"
	"time"
)

type XStream struct {
	stream
	sentContentLen int64
	conn           *XServerConn
	Request        *http.Request
	Response       *http.Response
	Trailer        *http.Header
	SendData       buffer.IoBuffer
	UseStream      bool
}

// ID returns stream id
func (ms *XStream) ID() uint32 {
	return ms.id
}

func (ms *XStream) WriteHeader(end bool) error {
	rsp := ms.Response
	endStream := end || ms.Request.Method == "HEAD"

	isHeadResp := ms.Request.Method == "HEAD"
	var ctype, clen string
	var dataLen int
	if ms.SendData != nil {
		dataLen = ms.SendData.Len()
	}

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

	ws := &writeResHeaders{
		streamID:      ms.id,
		httpResCode:   rsp.StatusCode,
		h:             rsp.Header,
		endStream:     endStream,
		contentLength: clen,
		contentType:   ctype,
		date:          date,
	}
	return ms.conn.writeHeaders(&ms.stream, ws)
}

func (ms *XStream) WriteData() (err error) {
	if ms.UseStream {
		var sawEOF bool
		bufp := buffer.GetBytes(defaultMaxReadFrameSize)
		defer buffer.PutBytes(bufp)
		buf := *bufp
		for !sawEOF {
			n, err := ms.SendData.Read(buf)
			if err == io.EOF {
				sawEOF = true
				err = nil
			} else if err != nil {
				return err
			}
			err = ms.conn.writeDataFromHandler(&ms.stream, buf[:n], false)
		}
	} else {
		err = ms.conn.writeDataFromHandler(&ms.stream, ms.SendData.Bytes(), false)
	}
	return
}

func (ms *XStream) WriteTrailers() error {
	var tramap http.Header
	if ms.Trailer != nil {
		tramap = *ms.Trailer
	}
	var trailers []string
	if tramap != nil {
		for k, _ := range tramap {
			k = http.CanonicalHeaderKey(k)
			if !httpguts.ValidTrailerHeader(k) {
				tramap.Del(k)
			} else {
				trailers = append(trailers, k)
			}
		}
	}

	var err error
	if len(trailers) > 0 {
		ws := &writeResHeaders{
			streamID:  ms.id,
			h:         tramap,
			trailers:  trailers,
			endStream: true,
		}
		err = ms.conn.writeHeaders(&ms.stream, ws)
	} else {
		err = ms.conn.writeDataFromHandler(&ms.stream, nil, true)

	}
	return err
}

func (ms *XStream) SendResponse() error {
	endHeader := ms.SendData == nil && ms.Trailer == nil
	if err := ms.WriteHeader(endHeader); err != nil || endHeader {
		return err
	}

	if err := ms.WriteData(); err != nil {
		return err
	}
	return ms.WriteTrailers()
}

type FrameHandler interface {
	HandleFrame(Frame, *XStream, []byte, bool, bool, error)
}

type XServerConn struct {
	serverConn
	preface  bool
	mHandler FrameHandler
}

func NewXServerConn(conn api.Connection, mHandler FrameHandler, opts *ServeConnOpts) *XServerConn {
	c := conn.RawConn()
	baseCtx, cancel := serverConnBaseContext(c, opts)
	defer cancel()

	sc := &XServerConn{
		serverConn: serverConn{
			srv:                         &Server{},
			hs:                          opts.baseConfig(),
			conn:                        c,
			baseCtx:                     baseCtx,
			remoteAddrStr:               c.RemoteAddr().String(),
			bw:                          newBufferedWriter(c),
			streams:                     make(map[uint32]*stream),
			readFrameCh:                 make(chan readFrameResult),
			wantWriteFrameCh:            make(chan FrameWriteRequest, 8),
			serveMsgCh:                  make(chan interface{}, 8),
			wroteFrameCh:                make(chan frameWriteResult, 1), // buffered; one send in writeFrameAsync
			bodyReadCh:                  make(chan bodyReadMsg),         // buffering doesn't matter either way
			doneServing:                 make(chan struct{}),
			clientMaxStreams:            math.MaxUint32, // Section 6.5.2: "Initially, there is no limit to this value"
			advMaxStreams:               defaultMaxStreams * 100,
			initialStreamSendWindowSize: initialWindowSize,
			maxFrameSize:                initialMaxFrameSize,
			headerTableSize:             initialHeaderTableSize,
			pushEnabled:                 true,
		},
		mHandler: mHandler,
	}

	// s.state.registerConn(sc)
	// defer s.state.unregisterConn(sc)

	// The net/http package sets the write deadline from the
	// http.Server.WriteTimeout during the TLS handshake, but then
	// passes the connection off to us with the deadline already set.
	// Write deadlines are set per stream in serverConn.newStream.
	// Disarm the net.Conn write deadline here.
	if sc.hs.WriteTimeout != 0 {
		sc.conn.SetWriteDeadline(time.Time{})
	}

	//if s.NewWriteScheduler != nil {
	//	sc.writeSched = s.NewWriteScheduler()
	//} else {
	//
	//}
	sc.writeSched = NewRandomWriteScheduler()

	// These start at the RFC-specified defaults. If there is a higher
	// configured value for inflow, that will be updated when we send a
	// WINDOW_UPDATE shortly after sending SETTINGS.
	sc.flow.add(initialWindowSize)
	sc.inflow.add(initialWindowSize)
	sc.hpackEncoder = hpack.NewEncoder(&sc.headerWriteBuf)

	fr := NewMFramer(conn)
	fr.ReadMetaHeaders = hpack.NewDecoder(initialHeaderTableSize, nil)
	fr.MaxHeaderListSize = sc.maxHeaderListSize()
	fr.SetMaxReadFrameSize(defaultMaxReadFrameSize)
	sc.framer = fr

	//if tc, ok := c.(connectionStater); ok {
	//	sc.tlsState = new(tls.ConnectionState)
	//	*sc.tlsState = tc.ConnectionState()
	//	// 9.2 Use of TLS Features
	//	// An implementation of HTTP/2 over TLS MUST use TLS
	//	// 1.2 or higher with the restrictions on feature set
	//	// and cipher suite described in this section. Due to
	//	// implementation limitations, it might not be
	//	// possible to fail TLS negotiation. An endpoint MUST
	//	// immediately terminate an HTTP/2 connection that
	//	// does not meet the TLS requirements described in
	//	// this section with a connection error (Section
	//	// 5.4.1) of type INADEQUATE_SECURITY.
	//	if sc.tlsState.Version < tls.VersionTLS12 {
	//		sc.rejectConn(ErrCodeInadequateSecurity, "TLS version too low")
	//		return
	//	}
	//
	//	if sc.tlsState.ServerName == "" {
	//		// Client must use SNI, but we don't enforce that anymore,
	//		// since it was causing problems when connecting to bare IP
	//		// addresses during development.
	//		//
	//		// TODO: optionally enforce? Or enforce at the time we receive
	//		// a new request, and verify the ServerName matches the :authority?
	//		// But that precludes proxy situations, perhaps.
	//		//
	//		// So for now, do nothing here again.
	//	}
	//
	//	if !s.PermitProhibitedCipherSuites && isBadCipher(sc.tlsState.CipherSuite) {
	//		// "Endpoints MAY choose to generate a connection error
	//		// (Section 5.4.1) of type INADEQUATE_SECURITY if one of
	//		// the prohibited cipher suites are negotiated."
	//		//
	//		// We choose that. In my opinion, the spec is weak
	//		// here. It also says both parties must support at least
	//		// TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 so there's no
	//		// excuses here. If we really must, we could allow an
	//		// "AllowInsecureWeakCiphers" option on the server later.
	//		// Let's see how it plays out first.
	//		sc.rejectConn(ErrCodeInadequateSecurity, fmt.Sprintf("Prohibited TLS 1.2 Cipher Suite: %x", sc.tlsState.CipherSuite))
	//		return
	//	}
	//}

	if hook := testHookGetServerConn; hook != nil {
		hook(&sc.serverConn)
	}
	go sc.serve()
	return sc
}

func (sc *XServerConn) serve() {
	sc.serveG = newGoroutineLock()
	defer sc.notePanic()
	defer sc.conn.Close()
	defer sc.closeAllStreamsOnConnClose()
	defer sc.stopShutdownTimer()
	defer close(sc.doneServing) // unblocks handlers trying to send

	if VerboseLogs {
		sc.vlogf("http2: server connection from %v on %p", sc.conn.RemoteAddr(), sc.hs)
	}

	sc.writeFrame(FrameWriteRequest{
		write: writeSettings{
			{SettingMaxFrameSize, sc.srv.maxReadFrameSize()},
			{SettingMaxConcurrentStreams, sc.advMaxStreams},
			{SettingMaxHeaderListSize, sc.maxHeaderListSize()},
			{SettingInitialWindowSize, uint32(sc.srv.initialStreamRecvWindowSize())},
		},
	})
	sc.unackedSettings++

	// Each connection starts with intialWindowSize inflow tokens.
	// If a higher value is configured, we add more tokens.
	if diff := sc.srv.initialConnRecvWindowSize() - initialWindowSize; diff > 0 {
		sc.sendWindowUpdate(nil, int(diff))
	}

	if sc.srv.IdleTimeout != 0 {
		sc.idleTimer = time.AfterFunc(sc.srv.IdleTimeout, sc.onIdleTimer)
		defer sc.idleTimer.Stop()
	}

	settingsTimer := time.AfterFunc(firstSettingsTimeout, sc.onSettingsTimer)
	defer settingsTimer.Stop()

	loopNum := 0
	for {
		loopNum++
		select {
		case wr := <-sc.wantWriteFrameCh:
			if se, ok := wr.write.(StreamError); ok {
				sc.resetStream(se)
				break
			}
			sc.writeFrame(wr)
		case res := <-sc.wroteFrameCh:
			sc.wroteFrame(res)
		case res := <-sc.readFrameCh:
			if !sc.processFrameFromReader(res) {
				return
			}
			res.readMore()
			if settingsTimer != nil {
				settingsTimer.Stop()
				settingsTimer = nil
			}
		case m := <-sc.bodyReadCh:
			sc.noteBodyRead(m.st, m.n)
		case msg := <-sc.serveMsgCh:
			switch v := msg.(type) {
			case func(int):
				v(loopNum) // for testing
			case *serverMessage:
				switch v {
				case settingsTimerMsg:
					sc.logf("timeout waiting for SETTINGS frames from %v", sc.conn.RemoteAddr())
					return
				case idleTimerMsg:
					sc.vlogf("connection is idle")
					sc.goAway(ErrCodeNo)
				case shutdownTimerMsg:
					sc.vlogf("GOAWAY close timer fired; closing conn from %v", sc.conn.RemoteAddr())
					return
				case gracefulShutdownMsg:
					sc.startGracefulShutdownInternal()
				default:
					panic("unknown timer")
				}
			case *startPushRequest:
				sc.startPush(v)
			default:
				panic(fmt.Sprintf("unexpected type %T", v))
			}
		}

		// Start the shutdown timer after sending a GOAWAY. When sending GOAWAY
		// with no error code (graceful shutdown), don't start the timer until
		// all open streams have been completed.
		sentGoAway := sc.inGoAway && !sc.needToSendGoAway && !sc.writingFrame
		gracefulShutdownComplete := sc.goAwayCode == ErrCodeNo && sc.curOpenStreams() == 0
		if sentGoAway && sc.shutdownTimer == nil && (sc.goAwayCode != ErrCodeNo || gracefulShutdownComplete) {
			sc.shutDownIn(goAwayTimeout)
		}
	}
}

// readPreface reads the ClientPreface greeting from the peer or
// returns errPrefaceTimeout on timeout, or an error if the greeting
// is invalid.
func (sc *XServerConn) readPreface(data types.IoBuffer) error {
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

func (sc *XServerConn) Decode(data types.IoBuffer, off int) error {
	if !sc.preface {
		if err := sc.readPreface(data); err != nil {
			sc.condlogf(err, "http2: server: error reading preface from client %v: %v", sc.conn.RemoteAddr(), err)
			return err
		}
		sc.preface = true
		// Now that we've got the preface, get us out of the
		// "StateNew" state. We can't go directly to idle, though.
		// Active means we read some data and anticipate a request. We'll
		// do another Active when we get a HEADERS frame.
		sc.setConnState(http.StateActive)
		sc.setConnState(http.StateIdle)
	}
	//f, _, err := sc.framer.(*XFramer).DecodeFrame(data, off)
	//if err == ErrAGAIN {
	//	return err
	//}
	//select {
	//case sc.readFrameCh <- readFrameResult{f, err, nil}:
	//case <-sc.doneServing:
	//	return errClientDisconnected
	//}
	gate := make(gate)
	gateDone := gate.Done
	f, _, err := sc.framer.(*XFramer).DecodeFrame(data, off)
	if err == ErrAGAIN {
		return err
	}
	select {
	case sc.readFrameCh <- readFrameResult{f, err, gateDone}:
	case <-sc.doneServing:
		return errClientDisconnected
	}
	select {
	case <-gate:
	case <-sc.doneServing:
		return errClientDisconnected
	}
	return err
}

// processFrameFromReader processes the serve loop's read from readFrameCh from the
// frame-reading goroutine.
// processFrameFromReader returns whether the connection should be kept open.
func (sc *XServerConn) processFrameFromReader(res readFrameResult) bool {
	sc.serveG.check()
	err := res.err
	if err != nil {
		if err == ErrFrameTooLarge {
			sc.goAway(ErrCodeFrameSize)
			return true // goAway will close the loop
		}
		clientGone := err == io.EOF || err == io.ErrUnexpectedEOF || isClosedConnError(err)
		if clientGone {
			// TODO: could we also get into this state if
			// the peer does a half close
			// (e.g. CloseWrite) because they're done
			// sending frames but they're still wanting
			// our open replies?  Investigate.
			// TODO: add CloseWrite to crypto/tls.Conn first
			// so we have a way to test this? I suppose
			// just for testing we could have a non-TLS mode.
			return false
		}
	} else {
		f := res.f
		if VerboseLogs {
			sc.vlogf("http2: server read frame %v", summarizeFrame(f))
		}
		err = sc.processFrame(f)
		if err == nil {
			return true
		}
	}

	switch ev := err.(type) {
	case StreamError:
		sc.resetStream(ev)
		return true
	case goAwayFlowError:
		sc.goAway(ErrCodeFlowControl)
		return true
	case ConnectionError:
		sc.logf("http2: server connection error from %v: %v", sc.conn.RemoteAddr(), ev)
		sc.goAway(ErrCode(ev))
		return true // goAway will handle shutdown
	default:
		if res.err != nil {
			sc.vlogf("http2: server closing client connection; error reading frame from client %s: %v", sc.conn.RemoteAddr(), err)
		} else {
			sc.logf("http2: server closing client connection: %v", err)
		}
		return false
	}
}

func (sc *XServerConn) processFrame(f Frame) error {
	sc.serveG.check()

	var err error
	var ms *XStream
	var endStream, trailer bool
	var data []byte

	// First frame received must be SETTINGS.
	if !sc.sawFirstSettings {
		if _, ok := f.(*SettingsFrame); !ok {
			return ConnectionError(ErrCodeProtocol)
		}
		sc.sawFirstSettings = true
	}

	switch f := f.(type) {
	case *SettingsFrame:
		err = sc.processSettings(f)
	case *MetaHeadersFrame:
		ms, trailer, endStream, err = sc.processHeaders(f)
	case *WindowUpdateFrame:
		err = sc.processWindowUpdate(f)
	case *PingFrame:
		err = sc.processPing(f)
	case *DataFrame:
		data = f.Data()
		endStream, err = sc.processData(f)
	case *RSTStreamFrame:
		err = sc.processResetStream(f)
	case *PriorityFrame:
		err = sc.processPriority(f)
	case *GoAwayFrame:
		err = sc.processGoAway(f)
	case *PushPromiseFrame:
		// A client cannot push. Thus, servers MUST treat the receipt of a PUSH_PROMISE
		// frame as a connection error (Section 5.4.1) of type PROTOCOL_ERROR.
		err = ConnectionError(ErrCodeProtocol)
	default:
		sc.vlogf("http2: server ignoring frame: %v", f.Header())
		return nil
	}

	sc.mHandler.HandleFrame(f, ms, data, endStream, trailer, err)
	return err
}

func (sc *XServerConn) HandleError(f Frame, err error) {
	if log.DefaultLogger.GetLogLevel() >= log.WARN {
		log.DefaultLogger.Warnf("[Server Conn] [Handler Err] handler frame：%v error：%v", f, err)
	}
}

func (sc *XServerConn) processHeaders(f *MetaHeadersFrame) (*XStream, bool, bool, error) {
	sc.serveG.check()
	id := f.StreamID
	if sc.inGoAway {
		// Ignore.
		return nil, false, false, nil
	}
	// http://tools.ietf.org/html/rfc7540#section-5.1.1
	// Streams initiated by a client MUST use odd-numbered stream
	// identifiers. [...] An endpoint that receives an unexpected
	// stream identifier MUST respond with a connection error
	// (Section 5.4.1) of type PROTOCOL_ERROR.
	if id%2 != 1 {
		return nil, false, false, ConnectionError(ErrCodeProtocol)
	}
	// A HEADERS frame can be used to create a new stream or
	// send a trailer for an open one. If we already have a stream
	// open, let it process its own HEADERS frame (trailers at this
	// point, if it's valid).
	if st := sc.streams[f.StreamID]; st != nil {
		if st.resetQueued {
			// We're sending RST_STREAM to close the stream, so don't bother
			// processing this frame.
			return nil, false, false, nil
		}
		// RFC 7540, sec 5.1: If an endpoint receives additional frames, other than
		// WINDOW_UPDATE, PRIORITY, or RST_STREAM, for a stream that is in
		// this state, it MUST respond with a stream error (Section 5.4.2) of
		// type STREAM_CLOSED.
		if st.state == stateHalfClosedRemote {
			return nil, false, false, streamError(id, ErrCodeStreamClosed)
		}
		return nil, true, true, st.processTrailerHeaders(f)
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

	if sc.idleTimer != nil {
		sc.idleTimer.Stop()
	}

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

	if f.HasPriority() {
		if err := checkPriority(f.StreamID, f.Priority); err != nil {
			return nil, false, false, err
		}
		sc.writeSched.AdjustStream(st.id, f.Priority)
	}

	_, req, err := sc.newWriterAndRequest(st, f)
	if err != nil {
		return nil, false, false, err
	}
	st.reqTrailer = req.Trailer
	if st.reqTrailer != nil {
		st.trailer = make(http.Header)
	}
	st.declBodyBytes = req.ContentLength

	// The net/http package sets the read deadline from the
	// http.Server.ReadTimeout during the TLS handshake, but then
	// passes the connection off to us with the deadline already
	// set. Disarm it here after the request headers are read,
	// similar to how the http1 server works. Here it's
	// technically more like the http1 Server's ReadHeaderTimeout
	// (in Go 1.8), though. That's a more sane option anyway.
	if sc.hs.ReadTimeout != 0 {
		sc.conn.SetReadDeadline(time.Time{})
	}

	return &XStream{stream: *st, Request: req, conn: sc}, false, f.StreamEnded(), nil
}

func (sc *XServerConn) processData(f *DataFrame) (bool, error) {
	sc.serveG.check()
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
		// RFC 7540, sec 8.1.2.6: A request or response is also malformed if the
		// value of a content-length header field does not equal the sum of the
		// DATA frame payload lengths that form the body.
		return false, streamError(id, ErrCodeProtocol)
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

		// Check the conn-level first, before the stream-level.
		if sc.inflow.available() < initialConnRecvWindowSize/2 {
			i := int(initialConnRecvWindowSize - sc.inflow.available())
			sc.sendWindowUpdate(nil, i)
		}

		if st.inflow.available() < initialConnRecvWindowSize/2 {
			i := int(initialConnRecvWindowSize - st.inflow.available())
			sc.sendWindowUpdate(st, i)
		}

		st.bodyBytes += int64(len(data))
	}
	if f.StreamEnded() {
		st.state = stateHalfClosedRemote
	}
	return f.StreamEnded(), nil
}
