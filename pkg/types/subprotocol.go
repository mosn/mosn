package types

type SubProtocol string

// 多路复用，Accesslog，流控，熔断能力
type Multiplexing interface {
	// 将Rpc数据包拆分为多个请求
	SplitFrame(data []byte) [][]byte
	GetStreamId(data []byte) string
	SetStreamId(data []byte,streamId string) []byte
}

// Tracing能力
type Tracing interface {
	Multiplexing
	GetServiceName(data []byte) string
	// tracing 是否method粒度?
	GetMethodName(data []byte) string
}

// RequestRouting, RequestAccessControl, RequesstFaultInjection能力
type RequestRouting interface {
	Multiplexing
	GetMetas(data []byte) map[string]string
}

// 协议转换能力
type ProtocolConvertor interface {
	Multiplexing
	Convert(data []byte) (map[string]string, []byte)
}