package types

const (
	HeaderStatus        = "x-mosn-status"
	HeaderMethod        = "x-mosn-method"
	HeaderHost          = "x-mosn-host"
	HeaderPath          = "x-mosn-path"
	HeaderStreamID      = "x-mosn-streamid"
	HeaderGlobalTimeout = "global-timeout"
	HeaderTryTimeout    = "try-timeout"
	HeaderException     = "x-mosn-exception"
)

const (
	UnSupportedProCode   string = " Protocol Code not supported"
	CodecException       string = "codec exception occurs"
	DeserializeException string = "deserial exception occurs"
)

const (
	CodecExceptionCode    int = 0
	UnknownCode           int = 2
	DeserialExceptionCode int = 3
	SuccessCode           int = 200
	RouterUnavailableCode int = 404
	NoHealthUpstreamCode  int = 500
	UpstreamOverFlowCode  int = 503
	TimeoutExceptionCode  int = 504
)
