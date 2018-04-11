package types

const (
	HeaderStatus          = "x-mosn-status"
	HeaderMethod          = "x-mosn-method"
	HeaderHost            = "x-mosn-host"
	HeaderPath            = "x-mosn-path"
	MosnStreamID          = "x-mosn-streamid"
	MosnGlobalTimeout     = "x-mosn-global-timeout"
	MosnTryTimeout        = "x-mosn-try-timeout"
	MosnExceptionCodeC    = "x-mosn-exception-codec"
	MosnExceptionDeserial = "x-mosn-exception-encode"
)

const (
	CodecExceptionCode    int = 0
	TimeoutExceptionCode  int = 1
	UnknownCode           int = 2
	RouterUnavailableCode int = 404
	NoHealthUpstreamCode  int = 500
	UpstreamOverFlowCode  int = 503
)

var ExceptionCodeArray = []string{MosnExceptionCodeC, MosnExceptionDeserial}



func CheckException(headers map[string]string) bool {

	for k, _ := range headers {
		for _, v := range ExceptionCodeArray {

			if k == v {
				return true
			}
		}

	}
	return false
}
