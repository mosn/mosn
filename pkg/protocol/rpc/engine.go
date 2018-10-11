package rpc

import (
	"github.com/alipay/sofa-mosn/pkg/types"
	"context"
	"github.com/alipay/sofa-mosn/pkg/log"
)

type engine struct {
	encoder types.Encoder
	decoder types.Decoder
}

type mixedEngine struct {
	engineMap map[byte]*engine
}

func NewEngine(encoder types.Encoder, decoder types.Decoder) types.ProtocolEngine {
	return &engine{
		encoder: encoder,
		decoder: decoder,
	}
}

func NewMixedEngine() types.ProtocolEngine {
	return &mixedEngine{
		engineMap: make(map[byte]*engine),
	}
}

func (eg *engine) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	return eg.encoder.Encode(ctx, model)
}

//func (eg *engine) EncodeTo(ctx context.Context, model interface{}, buf types.IoBuffer) (int, error) {
//	return eg.encoder.EncodeTo(ctx, model, buf)
//}

func (eg *engine) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	return eg.decoder.Decode(ctx, data)
}

func (eg *engine) Register(protocolCode byte, encoder types.Encoder, decoder types.Decoder) error {
	// unsupported for single protocol engine
	return nil
}

func (eg *engine) Process(ctx context.Context, data types.IoBuffer, handleFunc func(ctx2 context.Context, model interface{}, err error)) {

	for {
		cmd, err := eg.decoder.Decode(ctx, data)

		// No enough data
		if cmd == nil && err == nil {
			break;
		}

		// Do handle staff. Error would also be passed to this function.
		handleFunc(ctx, cmd, err)
	}
}

func (m *mixedEngine) Encode(ctx context.Context, model interface{}) (types.IoBuffer, error) {
	switch cmd := model.(type) {
	case RpcCmd:
		code := cmd.ProtocolCode()

		if eg, exists := m.engineMap[code]; exists {
			return eg.Encode(ctx, model)
		} else {
			return nil, ErrUnrecognizedCode
		}
	default:
		log.ByContext(ctx).Errorf("not RpcCmd, cannot find encoder for model = %+v", model)
		return nil, ErrUnknownType
	}
}

//func (m *mixedEngine) EncodeTo(ctx context.Context, model interface{}, buf types.IoBuffer) (int, error) {
//	switch cmd := model.(type) {
//	case RpcCmd:
//		code := cmd.ProtocolCode()
//
//		if eg, exists := m.engineMap[code]; exists {
//			return eg.EncodeTo(ctx, model, buf)
//		} else {
//			return 0, ErrUnrecognizedCode
//		}
//	default:
//
//		log.ByContext(ctx).Errorf("not RpcCmd, cannot find encoder for model = %+v", model)
//		return 0, ErrUnknownType
//	}
//}

func (m *mixedEngine) Decode(ctx context.Context, data types.IoBuffer) (interface{}, error) {
	// at least 1 byte for protocol code recognize
	if data.Len() > 1 {
		logger := log.ByContext(ctx)
		code := data.Bytes()[0]
		logger.Debugf("mixed protocol engine decode, protocol code = %x", code)

		if eg, exists := m.engineMap[code]; exists {
			return eg.Decode(ctx, data)
		} else {
			return nil, ErrUnrecognizedCode
		}
	}
	return nil, nil
}

func (m *mixedEngine) Register(protocolCode byte, encoder types.Encoder, decoder types.Decoder) error {
	// register engine
	if _, exists := m.engineMap[protocolCode]; exists {
		return ErrDupRegistered
	} else {
		m.engineMap[protocolCode] = &engine{
			encoder: encoder,
			decoder: decoder,
		}
	}
	return nil
}

func (m *mixedEngine) Process(ctx context.Context, data types.IoBuffer, handleFunc func(ctx2 context.Context, model interface{}, err error)) {
	// at least 1 byte for protocol code recognize
	for data.Len() > 1 {
		logger := log.ByContext(ctx)
		code := data.Bytes()[0]
		logger.Debugf("mixed protocol engine process, protocol code = %x", code)

		if eg, exists := m.engineMap[code]; exists {
			cmd, err := eg.Decode(ctx, data);
			// No enough data
			if cmd == nil && err == nil {
				break;
			}

			// Do handle staff. Error would also be passed to this function.
			handleFunc(ctx, cmd, err)
			if err != nil {
				return
			}
		} else {
			handleFunc(ctx, nil, ErrUnrecognizedCode)
			return
		}
	}
}
