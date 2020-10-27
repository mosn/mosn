package dubbo

import (
	"context"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/pkg/buffer"
	"reflect"
	"testing"
)

func TestFrame(t *testing.T) {
	baseContent := []byte("test")
	frame := &Frame{
		Header: Header{
			Magic:   MagicTag,
			Flag:    0x22,
			Status:  0x00,
			Id:      0,
			DataLen: 0,
		},
		payload: []byte{0x4e, 0x4e},
	}
	frame.SetRequestId(1)
	frame.Magic = MagicTag
	//not event
	frame.IsEvent = false
	//request
	frame.Direction = 1
	//set data
	frame.SetData(buffer.NewIoBufferBytes(baseContent))
	frame.Status = 0x14

	if frame.GetRequestId() != 1 {
		t.Errorf("dubbo method GetRequestId error")
	}
	if frame.IsHeartbeatFrame() != false {
		t.Errorf("dubbo method IsHeartbeatFrame error")
	}
	if frame.GetStreamType() != xprotocol.Request {
		t.Errorf("dubbo method GetStreamType error")
	}
	if frame.GetStatusCode() != 0x14 {
		t.Errorf("dubbo method GetStatusCode error")
	}
	content := frame.GetData().Bytes()
	if !reflect.DeepEqual(content, baseContent) {
		t.Errorf("dubbo method GetData error")
	}
	byteFrame, error := encodeFrame(nil, frame)
	if error != nil || !reflect.DeepEqual(byteFrame.Bytes(), encodeData) {
		t.Errorf("dubbo encode freame panic:%s", error)
	}
	//decode the byte data
	want, err := decodeFrame(context.TODO(), buffer.NewIoBufferBytes(encodeData))
	if err != nil {
		t.Errorf("dubbo decode freame panic:%s", err)
		return
	}
	wantFrame, ok := want.(*Frame)
	if !ok {
		t.Errorf("dubbo decode freame error")
		return
	}
	//compare the content
	if string(wantFrame.GetData().Bytes()) != "test" {
		t.Errorf("dubbo decode freame error, data not equal")
	}
}

var encodeData = []uint8{
	218, 187, 34, 20, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 4, 116, 101, 115, 116,
}
