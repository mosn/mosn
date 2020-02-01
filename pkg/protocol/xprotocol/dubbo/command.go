package dubbo

import (
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

type Header struct {
	Magic   []byte
	Flag    byte
	Status  byte
	Id      uint64
	DataLen uint32

	Event           int // 1 mean ping
	TwoWay          int // 1 mean req & resp pair
	Direction       int // 1 mean req
	SerializationId int // 2 mean hessian
}

type MetaMap struct {
	meta map[string]string
}

// ~ HeaderMap
func (metaMap *MetaMap) Get(key string) (string, bool) {
	return metaMap.Get(key)
}

func (metaMap *MetaMap) Set(key, value string) {
	metaMap.Set(key, value)
}

func (metaMap *MetaMap) Add(key, value string) {
	metaMap.Add(key, value)
}

func (metaMap *MetaMap) Del(key string) {
	metaMap.Del(key)
}

func (metaMap *MetaMap) Range(f func(key, value string) bool) {
	metaMap.Range(f)
}

func (metaMap *MetaMap) Clone() types.HeaderMap {
	newMetaMap := &MetaMap{
		meta: make(map[string]string, len(metaMap.meta)),
	}
	for k, v := range metaMap.meta {
		newMetaMap.meta[k] = v
	}
	return newMetaMap
}

func (metaMap *MetaMap) ByteSize() uint64 {
	size := 0
	for k, v := range metaMap.meta {
		size += len(k) + len(v)
	}
	return uint64(size)
}

type Frame struct {
	Header
	metaMap *MetaMap
	rawData []byte // raw data
	payload []byte // raw payload

	data    types.IoBuffer // wrapper of data
	content types.IoBuffer // wrapper of payload
}

// ~ XFrame
func (r *Frame) GetRequestId() uint64 {
	return r.Header.Id
}

func (r *Frame) SetRequestId(id uint64) {
	r.Header.Id = id
}

func (r *Frame) IsHeartbeatFrame() bool {
	return r.Header.Event != 0
}

func (r *Frame) GetStreamType() xprotocol.StreamType {
	switch r.Direction {
	case EventRequest:
		return xprotocol.Request
	case EventResponse:
		return xprotocol.Response
	default:
		return xprotocol.Request
	}
}

func (r *Frame) GetHeader() types.HeaderMap {
	return r.metaMap
}

func (r *Frame) GetData() types.IoBuffer {
	return r.content
}
