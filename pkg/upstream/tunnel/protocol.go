package tunnel

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"

	"mosn.io/api"
	"mosn.io/pkg/buffer"
)

var (
	Magic     = []byte{0x20, 0x88}
	HeaderLen = 4
	Version   = byte(0x01)

	MagicIndex   = 0
	VersionIndex = 2
	FlagIndex    = 3
	PayLoadIndex = 4

	typesToFlag = map[string]byte{
		reflect.TypeOf(ConnectionInitInfo{}).Name(): byte(0x01),
	}

	FlagToTypes = map[byte]func() interface{}{
		byte(0x01): func() interface{} {
			return &ConnectionInitInfo{}
		},
	}
)

type ConnectionInitInfo struct {
	ClusterName string `json:"cluster_name"`
	Weight      int64  `json:"weight"`
}

func WriteBuffer(i interface{}) (buffer.IoBuffer, error) {
	data, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	// alloc encode buffer
	totalLen := HeaderLen + len(data)
	buf := buffer.GetIoBuffer(totalLen)
	// encode header
	buf.Write(Magic)
	buf.WriteByte(Version)
	typeName := reflect.TypeOf(i).Elem().Name()
	if flag, ok := typesToFlag[typeName]; ok {
		buf.WriteByte(flag)
		// encode payload
		buf.Write(data)
		return buf, nil
	}
	return nil, errors.New("")
}

func Read(buffer api.IoBuffer) interface{} {
	dataBytes := buffer.Bytes()
	if len(dataBytes) == 0 {
		return nil
	}
	if len(dataBytes) <= HeaderLen {
		return nil
	}
	magicBytes := dataBytes[MagicIndex:VersionIndex]
	if bytes.Compare(Magic, magicBytes) != 0 {
		return nil
	}
	_ = dataBytes[VersionIndex:FlagIndex]
	flag := dataBytes[FlagIndex:]
	if f, ok := FlagToTypes[flag[0]]; ok {
		st := f()
		payload := dataBytes[PayLoadIndex:]
		err := json.Unmarshal(payload, st)
		if err != nil {
			return nil
		}
		buffer.Drain(buffer.Len())
		return st
	}

	return nil

}
