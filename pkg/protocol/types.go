package protocol

import "gitlab.alipay-inc.com/afe/mosn/pkg/types"

//统一的RPC PROTOCOL抽象接口
type Protocol interface {

	/**
	 * Get the encoder for the protocol.
	 *
	 * @return
	 */
	GetEncoder() types.Encoder

	/**
	 * Get the decoder for the protocol.
	 *
	 * @return
	 */
	GetDecoder() types.Decoder

	/**
	 * Get the heartbeat trigger for the protocol.
	 *
	 * @return
	 */
	//TODO
	//GetHeartbeatTrigger() HeartbeatTrigger

	/**
	 * Get the command handler for the protocol.
	 *
	 * @return
	 */
	//TODO
	//GetCommandHandler() CommandHandler
}

//TODO
type HeartbeatTrigger interface {
	HeartbeatTriggered()
}

//TODO
type CommandHandler interface {
	HandleCommand()
	RegisterProcessor()
	RegisterDefaultExecutor()
	GetDefaultExecutor()
}
