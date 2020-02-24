package dubbo

const (
	ProtocolName = "dubbo"
)

// dubbo protocol
const (
	HeaderLen   = 16
	IdLen       = 8
	MagicIdx    = 0
	FlagIdx     = 2
	StatusIdx   = 3
	IdIdx       = 4
	DataLenIdx  = 12
	DataLenSize = 4
)

// req/resp type
const (
	CmdTypeResponse      byte   = 0 // cmd type
	CmdTypeRequest       byte   = 1
	CmdTypeRequestOneway byte   = 2
	UnKnownCmdType       string = "unknown cmd type"
)

const (
	EventRequest  int = 1
	EventResponse int = 0
)

const (
	ServiceNameHeader string = "service"
	MethodNameHeader  string = "method"
)

const (
	ResponseStatusSuccess uint16 = 0x14 // 0x14 response status
)
