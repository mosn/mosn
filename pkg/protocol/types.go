package protocol


import "gitlab.alipay-inc.com/afe/mosn/pkg/types"



//统一的RPC PROTOCOL抽象接口
type Protocol interface{

	/**
	 * Get the encoder for the protocol.
	 *
	 * @return
	 */
	getEncoder()types.Encoder

	/**
	 * Get the decoder for the protocol.
	 *
	 * @return
	 */
	getDecoder()types.Decoder

	/**
	 * Get the heartbeat trigger for the protocol.
	 *
	 * @return
	 */
	getHeartbeatTrigger()HeartbeatTrigger

	/**
	 * Get the command handler for the protocol.
	 *
	 * @return
	 */

	getCommandHandler()CommandHandler

}

type HeartbeatTrigger interface{

	heartbeatTriggered()  //TODO

}


//TODO
type CommandHandler interface {

	handleCommand()
	registerProcessor()
	registerDefaultExecutor()
	getDefaultExecutor()

}


