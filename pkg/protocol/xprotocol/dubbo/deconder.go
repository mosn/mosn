package dubbo

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/AlexStocks/dubbogo/codec/hessian"
	"mosn.io/mosn/pkg/buffer"
	"mosn.io/mosn/pkg/types"
)

func decodeFrame(ctx context.Context, data types.IoBuffer) (cmd interface{}, err error) {
	// convert data to duboo frame
	dataBytes := data.Bytes()
	frame := &Frame{}
	// decode magic
	frame.Magic = dataBytes[MagicIdx:FlagIdx]
	// decode flag
	frame.Flag = dataBytes[FlagIdx]
	// decode status
	frame.Status = dataBytes[StatusIdx]
	// decode request id
	reqIdRaw := dataBytes[IdIdx:(IdIdx + IdLen)]
	frame.Id = binary.BigEndian.Uint64(reqIdRaw)
	// decode data length
	frame.DataLen = binary.BigEndian.Uint32(dataBytes[DataLenIdx:(DataLenIdx + DataLenSize)])

	// decode event
	eventBool := frame.Flag & (1 << 5)
	if eventBool != 0 {
		frame.Event = 1
	} else {
		frame.Event = 0
	}
	// decode twoway
	twoWayBool := frame.Flag & (1 << 6)
	if twoWayBool != 0 {
		frame.TwoWay = 1
	} else {
		frame.TwoWay = 0
	}
	// decode direction
	directionBool := frame.Flag & (1 << 7)
	if directionBool != 0 {
		frame.Direction = EventRequest
	} else {
		frame.Direction = EventResponse
	}
	// decode serializationId
	frame.SerializationId = int(frame.Flag & 0x1f)

	// decode payload
	payload := make([]byte, frame.DataLen)
	copy(payload, dataBytes[HeaderLen:HeaderLen+frame.DataLen])
	frame.payload = payload
	frame.content = buffer.NewIoBufferBytes(frame.payload)

	// not heartbeat & is request
	if frame.Event != 1 && frame.Direction == 1 {
		// service aware
		meta, err := getServiceAwareMeta(frame)
		if err != nil {
			return nil, err
		}
		for k, v := range meta {
			frame.Set(k, v)
		}
	}

	frameLen := HeaderLen + int(frame.DataLen)
	frame.rawData = dataBytes[:frameLen]
	frame.data = buffer.NewIoBufferBytes(frame.rawData)
	data.Drain(frameLen)
	return frame, nil
}

func getServiceAwareMeta(frame *Frame) (map[string]string, error) {
	meta := make(map[string]string)
	if frame.SerializationId != 2 {
		// not hessian , do not support
		return meta, fmt.Errorf("[xprotocol][dubbo] not hessian,do not support")
	}
	decoder := hessian.NewDecoder(frame.payload[:])
	var field interface{}
	var err error
	var ok bool
	var str string

	// dubbo version + path + version + method
	// get service name
	field, err = decoder.Decode()
	if err != nil {
		return meta, fmt.Errorf("[xprotocol][dubbo] decode version fail")
	}
	str, ok = field.(string)
	if !ok {
		return meta, fmt.Errorf("[xprotocol][dubbo] service name version type error")
	}

	field, err = decoder.Decode()
	if err != nil {
		return meta, fmt.Errorf("[xprotocol][dubbo] decode service fail")
	}
	str, ok = field.(string)
	if !ok {
		return meta, fmt.Errorf("[xprotocol][dubbo] service type error")
	}
	meta[ServiceNameHeader] = str

	// get method name
	field, err = decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("[xprotocol][dubbo] decode method version fail")
	}
	str, ok = field.(string)
	if !ok {
		return nil, fmt.Errorf("[xprotocol][dubbo] method version type fail")
	}

	field, err = decoder.Decode()
	if err != nil {
		return nil, fmt.Errorf("[xprotocol][dubbo] decode method fail")
	}
	str, ok = field.(string)
	if !ok {
		return nil, fmt.Errorf("[xprotocol][dubbo] method type error")
	}
	meta[MethodNameHeader] = str
	return meta, nil
}
