/******************************************************
# DESC    : network type & codec type
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-22 13:20
# FILE    : const.go
******************************************************/

package codec

//////////////////////////////////////////
// transport network
//////////////////////////////////////////

type TransportType int

const (
	TRANSPORTTYPE_BEGIN TransportType = iota
	TRANSPORT_TCP
	TRANSPORT_HTTP
	TRANSPORTTYPE_UNKNOWN
)

var transportTypeStrings = [...]string{
	"",
	"TCP",
	"HTTP",
	"",
}

func (t TransportType) String() string {
	if TRANSPORTTYPE_BEGIN < t && t < TRANSPORTTYPE_UNKNOWN {
		return transportTypeStrings[t]
	}

	return ""
}

func GetTransportType(t string) TransportType {
	var typ = TRANSPORTTYPE_UNKNOWN

	for i := TRANSPORTTYPE_BEGIN + 1; i < TRANSPORTTYPE_UNKNOWN; i++ {
		if transportTypeStrings[i] == t {
			typ = TransportType(i)
			break
		}
	}

	return typ
}

//////////////////////////////////////////
// codec type
//////////////////////////////////////////

type CodecType int

const (
	CODECTYPE_BEGIN CodecType = iota
	CODECTYPE_JSONRPC
	CODECTYPE_DUBBO
	CODECTYPE_UNKNOWN
)

var codecTypeStrings = [...]string{
	"",
	"jsonrpc",
	"dubbo",
	"",
}

func (c CodecType) String() string {
	if CODECTYPE_BEGIN < c && c < CODECTYPE_UNKNOWN {
		return codecTypeStrings[c]
	}

	return ""
}

func GetCodecType(t string) CodecType {
	var typ = CODECTYPE_UNKNOWN

	for i := CODECTYPE_BEGIN + 1; i < CODECTYPE_UNKNOWN; i++ {
		if codecTypeStrings[i] == t {
			typ = CodecType(i)
			break
		}
	}

	return typ
}
