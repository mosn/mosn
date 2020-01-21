package boltv2

import (
	"mosn.io/mosn/pkg/types"
)

// boltv2 constants
const (
	ProtocolName    types.ProtocolName = "boltv2" // protocol
	ProtocolCode    byte               = 2
	ProtocolVersion byte               = 2

	RequestHeaderLen  int = 24 // protocol header fields length
	ResponseHeaderLen int = 22
	LessLen           int = ResponseHeaderLen // minimal length for decoding

	RequestIdIndex         = 6
	RequestHeaderLenIndex  = 18
	ResponseHeaderLenIndex = 16
)
