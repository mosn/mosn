package tars

const (
	ProtocolName = "tars"
)

const (
	CmdTypeResponse byte   = 0 // cmd type
	CmdTypeRequest  byte   = 1
	CmdTypeUndefine byte   = 2
	UnKnownCmdType  string = "unknown cmd type"
)

const (
	ServiceNameHeader string = "service"
	MethodNameHeader  string = "method"
)
const (
	ResponseStatusSuccess uint16 = 0x00 // 0x00 response status
)
