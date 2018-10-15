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
