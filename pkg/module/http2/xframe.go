package http2

import (
	"encoding/binary"
	"errors"
	"golang.org/x/net/http/httpguts"
	"log"
	"mosn.io/api"
	"mosn.io/mosn/pkg/module/http2/hpack"
	"mosn.io/pkg/buffer"
	"strings"
)

type IFramer interface {
	WriteSettings(settings ...Setting) error
	WriteGoAway(maxStreamID uint32, code ErrCode, debugData []byte) error
	WriteData(streamID uint32, endStream bool, data []byte) error
	WriteRSTStream(streamID uint32, code ErrCode) error
	WritePing(ack bool, data [8]byte) error
	WriteSettingsAck() error
	WriteHeaders(p HeadersFrameParam) error
	WriteContinuation(streamID uint32, endHeaders bool, headerBlockFragment []byte) error
	WritePushPromise(p PushPromiseParam) error
	WriteWindowUpdate(streamID, incr uint32) error
	ReadFrame() (Frame, error)
}

type XFramer struct {
	Framer
	api.Connection
}

func NewMFramer(conn api.Connection) *XFramer {
	fr := &XFramer{
		Framer: Framer{
			logReads:          logFrameReads,
			logWrites:         logFrameWrites,
			debugReadLoggerf:  log.Printf,
			debugWriteLoggerf: log.Printf,
		},
		Connection: conn,
	}
	fr.SetMaxReadFrameSize(maxFrameSize)
	return fr
}

func (f *XFramer) startWrite(buf buffer.IoBuffer, ftype FrameType, flags Flags, streamID uint32) {
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

func (f *XFramer) endWrite(buf buffer.IoBuffer) error {
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
	return f.Connection.Write(buf)
}

func (f *XFramer) writeByte(b buffer.IoBuffer, v byte)     { b.Write([]byte{v}) }
func (f *XFramer) writeBytes(b buffer.IoBuffer, v []byte)  { b.Write(v) }
func (f *XFramer) writeUint16(b buffer.IoBuffer, v uint16) { b.Write([]byte{byte(v >> 8), byte(v)}) }
func (f *XFramer) writeUint32(b buffer.IoBuffer, v uint32) {
	b.Write([]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

func (fr *XFramer) ReadFrame() (Frame, error) {
	return nil, errors.New("unimplemented")
}

func mReadFrameHeader(data buffer.IoBuffer, off int) (FrameHeader, error) {
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

func (fr *XFramer) DecodeFrame(data buffer.IoBuffer, off int) (Frame, int, error) {
	fr.errDetail = nil
	last := fr.lastFrame
	lastHeader := fr.lastHeaderStream
	fh, err := mReadFrameHeader(data, off)
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
		f, msize, err = fr.readMetaFrame(f.(*HeadersFrame), data, off+size)

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

// WriteData writes a DATA frame.
//
// It will perform exactly one Write to the underlying Writer.
// It is the caller's responsibility not to violate the maximum frame size
// and to not call other Write methods concurrently.
func (f *XFramer) WriteData(streamID uint32, endStream bool, data []byte) error {
	return f.WriteDataPadded(streamID, endStream, data, nil)
}

// WriteData writes a DATA frame with optional padding.
//
// If pad is nil, the padding bit is not sent.
// The length of pad must not exceed 255 bytes.
// The bytes of pad must all be zero, unless f.AllowIllegalWrites is set.
//
// It will perform exactly one Write to the underlying Writer.
// It is the caller's responsibility not to violate the maximum frame size
// and to not call other Write methods concurrently.
func (f *XFramer) WriteDataPadded(streamID uint32, endStream bool, data, pad []byte) error {
	if !validStreamID(streamID) && !f.AllowIllegalWrites {
		return errStreamID
	}
	if len(pad) > 0 {
		if len(pad) > 255 {
			return errPadLength
		}
		if !f.AllowIllegalWrites {
			for _, b := range pad {
				if b != 0 {
					// "Padding octets MUST be set to zero when sending."
					return errPadBytes
				}
			}
		}
	}
	var flags Flags
	if endStream {
		flags |= FlagDataEndStream
	}
	if pad != nil {
		flags |= FlagDataPadded
	}
	buf := buffer.GetIoBuffer(len(data) + frameHeaderLen + 8)
	f.startWrite(buf, FrameData, flags, streamID)
	if pad != nil {
		f.writeByte(buf, byte(len(pad)))
	}
	buf.Write(data)
	buf.Write(pad)
	return f.endWrite(buf)
}

// WriteSettings writes a SETTINGS frame with zero or more settings
// specified and the ACK bit not set.
//
// It will perform exactly one Write to the underlying Writer.
// It is the caller's responsibility to not call other Write methods concurrently.
func (f *XFramer) WriteSettings(settings ...Setting) error {
	buf := buffer.NewIoBuffer(len(settings)*6 + frameHeaderLen)
	f.startWrite(buf, FrameSettings, 0, 0)
	for _, s := range settings {
		f.writeUint16(buf, uint16(s.ID))
		f.writeUint32(buf, s.Val)
	}
	return f.endWrite(buf)
}

// WriteSettingsAck writes an empty SETTINGS frame with the ACK bit set.
//
// It will perform exactly one Write to the underlying Writer.
// It is the caller's responsibility to not call other Write methods concurrently.
func (f *XFramer) WriteSettingsAck() error {
	buf := buffer.NewIoBuffer(frameHeaderLen)
	f.startWrite(buf, FrameSettings, FlagSettingsAck, 0)
	return f.endWrite(buf)
}

func (f *XFramer) WritePing(ack bool, data [8]byte) error {
	var flags Flags
	if ack {
		flags = FlagPingAck
	}
	buf := buffer.NewIoBuffer(len(data[:]) + frameHeaderLen)
	f.startWrite(buf, FramePing, flags, 0)
	f.writeBytes(buf, data[:])
	return f.endWrite(buf)
}

func (f *XFramer) WriteGoAway(maxStreamID uint32, code ErrCode, debugData []byte) error {
	buf := buffer.NewIoBuffer(8 + len(debugData) + frameHeaderLen)
	f.startWrite(buf, FrameGoAway, 0, 0)
	f.writeUint32(buf, maxStreamID&(1<<31-1))
	f.writeUint32(buf, uint32(code))
	f.writeBytes(buf, debugData)
	return f.endWrite(buf)
}

// WriteWindowUpdate writes a WINDOW_UPDATE frame.
// The increment value must be between 1 and 2,147,483,647, inclusive.
// If the Stream ID is zero, the window update applies to the
// connection as a whole.
func (f *XFramer) WriteWindowUpdate(streamID, incr uint32) error {
	// "The legal range for the increment to the flow control window is 1 to 2^31-1 (2,147,483,647) octets."
	if (incr < 1 || incr > 2147483647) && !f.AllowIllegalWrites {
		return errors.New("illegal window increment value")
	}
	buf := buffer.NewIoBuffer(4 + frameHeaderLen)
	f.startWrite(buf, FrameWindowUpdate, 0, streamID)
	f.writeUint32(buf, incr)
	return f.endWrite(buf)
}

// WriteHeaders writes a single HEADERS frame.
//
// This is a low-level header writing method. Encoding headers and
// splitting them into any necessary CONTINUATION frames is handled
// elsewhere.
//
// It will perform exactly one Write to the underlying Writer.
// It is the caller's responsibility to not call other Write methods concurrently.
func (f *XFramer) WriteHeaders(p HeadersFrameParam) error {
	if !validStreamID(p.StreamID) && !f.AllowIllegalWrites {
		return errStreamID
	}
	var flags Flags
	buf := buffer.GetIoBuffer(len(p.BlockFragment) + frameHeaderLen + 8)
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
	f.startWrite(buf, FrameHeaders, flags, p.StreamID)
	if p.PadLength != 0 {
		f.writeByte(buf, p.PadLength)
	}
	if !p.Priority.IsZero() {
		v := p.Priority.StreamDep
		if !validStreamIDOrZero(v) && !f.AllowIllegalWrites {
			return errDepStreamID
		}
		if p.Priority.Exclusive {
			v |= 1 << 31
		}
		f.writeUint32(buf, v)
		f.writeByte(buf, p.Priority.Weight)
	}
	buf.Write(p.BlockFragment)
	buf.Write(padZeros[:p.PadLength])
	return f.endWrite(buf)
}

// WritePriority writes a PRIORITY frame.
//
// It will perform exactly one Write to the underlying Writer.
// It is the caller's responsibility to not call other Write methods concurrently.
func (f *XFramer) WritePriority(streamID uint32, p PriorityParam) error {
	if !validStreamID(streamID) && !f.AllowIllegalWrites {
		return errStreamID
	}
	if !validStreamIDOrZero(p.StreamDep) {
		return errDepStreamID
	}
	buf := buffer.GetIoBuffer(5 + frameHeaderLen)
	f.startWrite(buf, FramePriority, 0, streamID)
	v := p.StreamDep
	if p.Exclusive {
		v |= 1 << 31
	}
	f.writeUint32(buf, v)
	f.writeByte(buf, p.Weight)
	return f.endWrite(buf)
}

// WriteRSTStream writes a RST_STREAM frame.
//
// It will perform exactly one Write to the underlying Writer.
// It is the caller's responsibility to not call other Write methods concurrently.
func (f *XFramer) WriteRSTStream(streamID uint32, code ErrCode) error {
	if !validStreamID(streamID) && !f.AllowIllegalWrites {
		return errStreamID
	}
	buf := buffer.GetIoBuffer(4 + frameHeaderLen)
	f.startWrite(buf, FrameRSTStream, 0, streamID)
	f.writeUint32(buf, uint32(code))
	return f.endWrite(buf)
}

// WriteContinuation writes a CONTINUATION frame.
//
// It will perform exactly one Write to the underlying Writer.
// It is the caller's responsibility to not call other Write methods concurrently.
func (f *XFramer) WriteContinuation(streamID uint32, endHeaders bool, headerBlockFragment []byte) error {
	if !validStreamID(streamID) && !f.AllowIllegalWrites {
		return errStreamID
	}
	var flags Flags
	if endHeaders {
		flags |= FlagContinuationEndHeaders
	}
	buf := buffer.GetIoBuffer(len(headerBlockFragment) + frameHeaderLen)
	f.startWrite(buf, FrameContinuation, flags, streamID)
	buf.Write(headerBlockFragment)
	return f.endWrite(buf)
}

// WritePushPromise writes a single PushPromise Frame.
//
// As with Header Frames, This is the low level call for writing
// individual frames. Continuation frames are handled elsewhere.
//
// It will perform exactly one Write to the underlying Writer.
// It is the caller's responsibility to not call other Write methods concurrently.
func (f *XFramer) WritePushPromise(p PushPromiseParam) error {
	if !validStreamID(p.StreamID) && !f.AllowIllegalWrites {
		return errStreamID
	}
	var flags Flags
	if p.PadLength != 0 {
		flags |= FlagPushPromisePadded
	}
	if p.EndHeaders {
		flags |= FlagPushPromiseEndHeaders
	}
	buf := buffer.GetIoBuffer(len(p.BlockFragment) + frameHeaderLen + 12)
	f.startWrite(buf, FramePushPromise, flags, p.StreamID)
	if p.PadLength != 0 {
		f.writeByte(buf, p.PadLength)
	}
	if !validStreamID(p.PromiseID) && !f.AllowIllegalWrites {
		return errStreamID
	}
	f.writeUint32(buf, p.PromiseID)
	buf.Write(p.BlockFragment)
	buf.Write(padZeros[:p.PadLength])
	return f.endWrite(buf)
}

// WriteRawFrame writes a raw frame. This can be used to write
// extension frames unknown to this package.
func (f *XFramer) WriteRawFrame(t FrameType, flags Flags, streamID uint32, payload []byte) error {
	buf := buffer.GetIoBuffer(len(payload) + frameHeaderLen)
	f.startWrite(buf, t, flags, streamID)
	f.writeBytes(buf, payload)
	return f.endWrite(buf)
}

// readMetaFrame returns 0 or more CONTINUATION frames from fr and
// merge them into the provided hf and returns a MetaHeadersFrame
// with the decoded hpack values.
func (fr *XFramer) readMetaFrame(hf *HeadersFrame, data buffer.IoBuffer, off int) (*MetaHeadersFrame, int, error) {
	if fr.AllowIllegalReads {
		return nil, 0, errors.New("illegal use of AllowIllegalReads with ReadMetaHeaders")
	}
	mh := &MetaHeadersFrame{
		HeadersFrame: hf,
	}
	var remainSize = fr.maxHeaderListSize()
	var sawRegular bool

	var invalid error // pseudo header field errors
	hdec := fr.ReadMetaHeaders
	hdec.SetEmitEnabled(true)
	hdec.SetMaxStringLength(fr.maxHeaderStringLen())
	hdec.SetEmitFunc(func(hf hpack.HeaderField) {
		if VerboseLogs && fr.logReads {
			fr.debugReadLoggerf("http2: decoded hpack field %+v", hf)
		}
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

	var hc headersOrContinuation = hf
	msize := 0
	for {
		frag := hc.HeaderBlockFragment()
		if _, err := hdec.Write(frag); err != nil {
			return nil, 0, ConnectionError(ErrCodeCompression)
		}

		if hc.HeadersEnded() {
			break
		}
		if f, size, err := fr.DecodeFrame(data, off); err != nil {
			return nil, 0, err
		} else {
			msize += size
			hc = f.(*ContinuationFrame) // guaranteed by checkFrameOrder
		}
	}

	mh.HeadersFrame.headerFragBuf = nil
	mh.HeadersFrame.invalidate()

	if err := hdec.Close(); err != nil {
		return nil, 0, ConnectionError(ErrCodeCompression)
	}
	if invalid != nil {
		fr.errDetail = invalid
		if VerboseLogs {
			log.Printf("http2: invalid header: %v", invalid)
		}
		return nil, 0, StreamError{mh.StreamID, ErrCodeProtocol, invalid}
	}
	if err := mh.checkPseudos(); err != nil {
		fr.errDetail = err
		if VerboseLogs {
			log.Printf("http2: invalid pseudo headers: %v", err)
		}
		return nil, 0, StreamError{mh.StreamID, ErrCodeProtocol, err}
	}
	return mh, msize, nil
}
