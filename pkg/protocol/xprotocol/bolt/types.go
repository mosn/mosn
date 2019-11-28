package bolt

import (
	"time"
	"errors"
	"sofastack.io/sofa-mosn/pkg/api/v2"
	"sofastack.io/sofa-mosn/pkg/types"
)

// bolt constants
const (
	ProtocolName    types.ProtocolName = "Bolt" // protocol
	ProtocolCode    byte               = 1
	ProtocolVersion byte               = 1

	CmdTypeResponse      byte = 0 // cmd type
	CmdTypeRequest       byte = 1
	CmdTypeRequestOneway byte = 2

	CmdCodeHeartbeat   uint16 = 0 // cmd code
	CmdCodeRpcRequest  uint16 = 1
	CmdCodeRpcResponse uint16 = 2

	Hessian2Serialize byte = 1 // serialize

	ResponseStatusSuccess                 uint16 = 0  // 0x00 response status
	ResponseStatusError                   uint16 = 1  // 0x01
	ResponseStatusServerException         uint16 = 2  // 0x02
	ResponseStatusUnknown                 uint16 = 3  // 0x03
	ResponseStatusServerThreadpoolBusy    uint16 = 4  // 0x04
	ResponseStatusErrorComm               uint16 = 5  // 0x05
	ResponseStatusNoProcessor             uint16 = 6  // 0x06
	ResponseStatusTimeout                 uint16 = 7  // 0x07
	ResponseStatusClientSendError         uint16 = 8  // 0x08
	ResponseStatusCodecException          uint16 = 9  // 0x09
	ResponseStatusConnectionClosed        uint16 = 16 // 0x10
	ResponseStatusServerSerialException   uint16 = 17 // 0x11
	ResponseStatusServerDeserialException uint16 = 18 // 0x12

	RequestHeaderLen  int = 22 // protocol header fields length
	ResponseHeaderLen int = 20
	LessLen           int = ResponseHeaderLen // minimal length for decoding

	RequestIdIndex         = 5
	RequestHeaderLenIndex  = 16
	ResponseHeaderLenIndex = 14
)

const (
	// Encode/Decode Exception Msg
	UnKnownCmdType string = "unknown cmd type"
	UnKnownCmdCode string = "unknown cmd code"

	// Sofa Rpc Default HC Parameters
	SofaRPC                             = "SofaRpc"
	DefaultBoltHeartBeatTimeout         = 6 * 15 * time.Second
	DefaultBoltHeartBeatInterval        = 15 * time.Second
	DefaultIntervalJitter               = 5 * time.Millisecond
	DefaultHealthyThreshold      uint32 = 2
	DefaultUnhealthyThreshold    uint32 = 2
)

var (
	// Encode/Decode Exception
	ErrUnKnownCmdType = errors.New(UnKnownCmdType)
	ErrUnKnownCmdCode = errors.New(UnKnownCmdCode)
)

// DefaultSofaRPCHealthCheckConf
var DefaultSofaRPCHealthCheckConf = v2.HealthCheck{
	HealthCheckConfig: v2.HealthCheckConfig{
		Protocol:           SofaRPC,
		HealthyThreshold:   DefaultHealthyThreshold,
		UnhealthyThreshold: DefaultUnhealthyThreshold,
	},
	Timeout:        DefaultBoltHeartBeatTimeout,
	Interval:       DefaultBoltHeartBeatInterval,
	IntervalJitter: DefaultIntervalJitter,
}
