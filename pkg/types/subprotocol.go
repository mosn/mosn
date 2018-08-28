package types

// SubProtocol Name
type SubProtocol string

// Multiplexing Accesslog Rate limit Curcuit Breakers
type Multiplexing interface {
	SplitFrame(data []byte) [][]byte
	GetStreamID(data []byte) string
	SetStreamID(data []byte, streamID string) []byte
}

// Tracing base on Multiplexing
type Tracing interface {
	Multiplexing
	GetServiceName(data []byte) string
	GetMethodName(data []byte) string
}

// RequestRouting RequestAccessControl RequesstFaultInjection base on Multiplexing
type RequestRouting interface {
	Multiplexing
	GetMetas(data []byte) map[string]string
}

// ProtocolConvertor change protocol base on Multiplexing
type ProtocolConvertor interface {
	Multiplexing
	Convert(data []byte) (map[string]string, []byte)
}
