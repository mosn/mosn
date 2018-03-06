package codec

//command defination
type trCommand struct {
	id        int
	protocol  byte
	direction byte

	cmdCode byte

	conBytes     []byte
	packet       []byte
	appClassName string
	totalLength  int

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

