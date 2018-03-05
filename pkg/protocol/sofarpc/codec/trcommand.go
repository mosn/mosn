package codec

//command defination
type TrCommand struct {
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

type TrRequestCommand struct {
	//rpc command
	TrCommand

	//request command
	request interface{} //TODO import ConnectionRequest
}

type TRResponseCommand struct {
	//rpc command
	TrCommand

	//response command
	response interface{} //TODO import ConnectionResponse
}
