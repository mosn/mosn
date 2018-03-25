package codec

//command defination
type trCommand struct {
	id              int
	protocol        byte
	protocolVersion byte
	requestFlag     byte

	direction byte

	//请求,心跳等
	cmdCode int16

	conBytes     []byte
	packet       []byte
	appClassName string

	connRequestContent []byte

	appClassContent []byte
	totalLength     uint32

	//存储原始的 byte
	originalBytes []byte
	invokeContext interface{}
}

// tr request cmd
type trRequestCommand struct {
	//rpc command
	trCommand

	//request command
	request interface{} //TODO import ConnectionRequest
}

// tr resp cmd
type trResponseCommand struct {
	//rpc command
	trCommand

	//response command
	response interface{} //TODO import ConnectionResponse
}
