package types

type SubProtocol string

// Multiplexing, Accesslog, Rate limit, Curcuit Breakers
type Multiplexing interface {
	SplitFrame(data []byte) [][]byte
	GetStreamId(data []byte) string
	SetStreamId(data []byte, streamId string) []byte
}

// Tracing
type Tracing interface {
	Multiplexing
	GetServiceName(data []byte) string
	GetMethodName(data []byte) string
}

// RequestRouting, RequestAccessControl, RequesstFaultInjection
type RequestRouting interface {
	Multiplexing
	GetMetas(data []byte) map[string]string
}

// Protocol convert
type ProtocolConvertor interface {
	Multiplexing
	Convert(data []byte) (map[string]string, []byte)
}
