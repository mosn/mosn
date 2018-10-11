package rpc

import (
	"github.com/alipay/sofa-mosn/pkg/types"

	"errors"
)

var (
	AlreadyRegistered = "protocol code already registered."
	UnknownType       = "unknown model type."
	UnrecognizedCode  = "unrecognized protocol code."
	NoProtocolCode    = "no protocol code found."

	ErrDupRegistered    = errors.New(AlreadyRegistered)
	ErrUnknownType      = errors.New(UnknownType)
	ErrUnrecognizedCode = errors.New(UnrecognizedCode)
	ErrNoProtocolCode   = errors.New(NoProtocolCode)
)

// RpcCmd act as basic model for different protocols
type RpcCmd interface {
	types.HeaderMap

	ProtocolCode() byte

	RequestID() uint32

	SetRequestID(requestID uint32)

	Header() map[string]string

	Data() []byte

	SetHeader(header map[string]string)

	SetData(data []byte)
}

// SofaRpcCmd  act as basic model for sofa protocols
type SofaRpcCmd interface {
	RpcCmd

	CommandType() byte

	CommandCode() int16
}

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
