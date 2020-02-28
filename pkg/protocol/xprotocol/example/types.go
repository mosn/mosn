package example

import (
	"mosn.io/mosn/pkg/types"
)

// protocol constants
const (
	ProtocolName types.ProtocolName = "x_example" // protocol

	Magic       byte = 'x' //magic
	DirRequest  byte = 0   // dir
	DirResponse byte = 1   // dir

	TypeHeartbeat byte = 0 // cmd code
	TypeMessage   byte = 1
	TypeGoAway    byte = 2

	ResponseStatusSuccess uint16 = 0 // 0x00 response status
	ResponseStatusError   uint16 = 1 // 0x01

	RequestHeaderLen  int = 11 // protocol header fields length
	ResponseHeaderLen int = 13
	MinimalDecodeLen  int = RequestHeaderLen // minimal length for decoding

	RequestIdIndex = 3
)
