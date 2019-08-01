package mirror

import (
	"sofastack.io/sofa-mosn/pkg/buffer"
	"sofastack.io/sofa-mosn/pkg/types"
)

const (
	ClassSeparator    = 0x01
	DataSeparator     = 0x02
	KeyValueSeparator = 0x03
	End               = 0x00
)

type MirrorData struct {
	Protocol byte
	// Binarys contains [{request_body}, {response_body}]
	Binarys [2]types.IoBuffer
	// KeyValues contains [{request_header}, {request_trailer}, {response_header},{response_trailer}]
	KeyValues [4]types.HeaderMap
	// Strings contains information (with sequence):
	// {traceid},{callerApp},{TargetApp},{InterfaceName},{MethodName},{Src IP},{Dst IP},{Dst Port}
	Strings [8]string
}

func (md *MirrorData) WriteProtocol(p byte) {
	md.Protocol = p
}

func (md *MirrorData) WriteRequest(header types.HeaderMap, body types.IoBuffer, trailer types.HeaderMap) {
	md.Binarys[0] = body
	md.KeyValues[0] = header
	md.KeyValues[1] = trailer
}

func (md *MirrorData) WriteResponse(header types.HeaderMap, body types.IoBuffer, trailer types.HeaderMap) {
	md.Binarys[1] = body
	md.KeyValues[2] = header
	md.KeyValues[3] = trailer
}

func (md *MirrorData) WriteTraceID(s string) {
	md.Strings[0] = s
}

func (md *MirrorData) WriteCallerApp(s string) {
	md.Strings[1] = s
}

func (md *MirrorData) WriteTargetApp(s string) {
	md.Strings[2] = s
}

func (md *MirrorData) WriteInterface(s string) {
	md.Strings[3] = s
}

func (md *MirrorData) WriteMethod(s string) {
	md.Strings[4] = s
}

func (md *MirrorData) WriteSrcIP(s string) {
	md.Strings[5] = s
}

func (md *MirrorData) WriteDstIP(s string) {
	md.Strings[6] = s
}

func (md *MirrorData) WriteDstPort(s string) {
	md.Strings[7] = s
}

func (md *MirrorData) Buffer() types.IoBuffer {
	buf := buffer.GetIoBuffer(1024)
	buf.Write([]byte{md.Protocol, ClassSeparator})
}
