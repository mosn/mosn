package bolt

const (
	// ~~ header name of protocol field
	HeaderProtocolCode  string = "protocol"
	HeaderCmdType       string = "cmdtype"
	HeaderCmdCode       string = "cmdcode"
	HeaderVersion       string = "version"
	HeaderReqID         string = "requestid"
	HeaderCodec         string = "codec"
	HeaderTimeout       string = "timeout"
	HeaderClassLen      string = "classlen"
	HeaderHeaderLen     string = "headerlen"
	HeaderContentLen    string = "contentlen"
	HeaderClassName     string = "classname"
	HeaderVersion1      string = "ver1"
	HeaderSwitchCode    string = "switchcode"
	HeaderRespStatus    string = "respstatus"
	HeaderRespTimeMills string = "resptimemills"

	// ~~ constans
	PROTOCOL_CODE_V1 byte = 1 // protocol code
	PROTOCOL_CODE_V2 byte = 2

	PROTOCOL_VERSION_1 byte = 1 // version
	PROTOCOL_VERSION_2 byte = 2

	REQUEST_HEADER_LEN_V1 int = 22 // protocol header fields length
	REQUEST_HEADER_LEN_V2 int = 24

	RESPONSE_HEADER_LEN_V1 int = 20
	RESPONSE_HEADER_LEN_V2 int = 22

	LESS_LEN_V1 int = RESPONSE_HEADER_LEN_V1 // minimal length for decoding
	LESS_LEN_V2 int = RESPONSE_HEADER_LEN_V2

	RESPONSE       byte = 0 // cmd type
	REQUEST        byte = 1
	REQUEST_ONEWAY byte = 2

	HEARTBEAT    int16 = 0 // cmd code
	RPC_REQUEST  int16 = 1
	RPC_RESPONSE int16 = 2

	HESSIAN2_SERIALIZE byte = 1 // serialize

	RESPONSE_STATUS_SUCCESS                   int16 = 0  // 0x00 response status
	RESPONSE_STATUS_ERROR                     int16 = 1  // 0x01
	RESPONSE_STATUS_SERVER_EXCEPTION          int16 = 2  // 0x02
	RESPONSE_STATUS_UNKNOWN                   int16 = 3  // 0x03
	RESPONSE_STATUS_SERVER_THREADPOOL_BUSY    int16 = 4  // 0x04
	RESPONSE_STATUS_ERROR_COMM                int16 = 5  // 0x05
	RESPONSE_STATUS_NO_PROCESSOR              int16 = 6  // 0x06
	RESPONSE_STATUS_TIMEOUT                   int16 = 7  // 0x07
	RESPONSE_STATUS_CLIENT_SEND_ERROR         int16 = 8  // 0x08
	RESPONSE_STATUS_CODEC_EXCEPTION           int16 = 9  // 0x09
	RESPONSE_STATUS_CONNECTION_CLOSED         int16 = 16 // 0x10
	RESPONSE_STATUS_SERVER_SERIAL_EXCEPTION   int16 = 17 // 0x11
	RESPONSE_STATUS_SERVER_DESERIAL_EXCEPTION int16 = 18 // 0x12
)
