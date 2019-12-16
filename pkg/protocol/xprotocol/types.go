package xprotocol

import (
	"sofastack.io/sofa-mosn/pkg/types"
	"errors"
)

// XStream
type StreamType int

const (
	Request       StreamType = iota
	RequestOneWay
	Response
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

type XFrame interface {
	Multiplexing

	HeartbeatPredicate

	GetStreamType() StreamType

	GetHeader() types.HeaderMap

	GetData() types.IoBuffer
}

type Multiplexing interface {
	GetRequestId() uint64

	SetRequestId(id uint64)
}

type HeartbeatPredicate interface {
	IsHeartbeatFrame() bool
}

type HeaderMutator interface {
	MutateHeader(bytes []byte) []byte
}

type ServiceAware interface {
	GetServiceName() string

	GetMethodName() string
}

type GoAwayPredicate interface {
	IsGoAwayFrame() bool
}

// XProtocol
type XProtocol interface {
	types.Protocol

	Heartbeater

	Hijacker
}

// HeartbeatBuilder provides interface to construct proper heartbeat command for xprotocol sub-protocols
type Heartbeater interface {
	// Trigger builds an active heartbeat command
	Trigger(requestId uint64) XFrame

	// Reply builds heartbeat command corresponding to the given requestID
	Reply(requestId uint64) XFrame
}

// HeartbeatBuilder provides interface to construct proper response command for xprotocol sub-protocols
type Hijacker interface {
	// BuildResponse build response with given status code
	Hijack(statusCode uint32) XFrame

	Mapping(httpStatusCode uint32) uint32
}
