package tars

import (
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	"mosn.io/mosn/pkg/protocol/xprotocol"
	"mosn.io/mosn/pkg/types"
)

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

type Request struct {
	cmd     *requestf.RequestPacket
	metaMap *MetaMap
	rawData []byte         // raw data
	data    types.IoBuffer // wrapper of data
}

// ~ XFrame
func (r *Request) GetRequestId() uint64 {
	return uint64(r.cmd.IRequestId)
}

func (r *Request) SetRequestId(id uint64) {
	r.cmd.IRequestId = int32(id)
}

func (r *Request) IsHeartbeatFrame() bool {
	// un support
	return false
}

func (r *Request) GetStreamType() xprotocol.StreamType {
	return xprotocol.Request
}

func (r *Request) GetHeader() types.HeaderMap {
	return r.metaMap
}

func (r *Request) GetData() types.IoBuffer {
	return r.data
}

type Response struct {
	cmd     *requestf.ResponsePacket
	metaMap *MetaMap
	rawData []byte         // raw data
	data    types.IoBuffer // wrapper of data
}

// ~ XFrame
func (r *Response) GetRequestId() uint64 {
	return uint64(r.cmd.IRequestId)
}

func (r *Response) SetRequestId(id uint64) {
	r.cmd.IRequestId = int32(id)
}

func (r *Response) IsHeartbeatFrame() bool {
	// un support
	return false
}

func (r *Response) GetStreamType() xprotocol.StreamType {
	return xprotocol.Request
}

func (r *Response) GetHeader() types.HeaderMap {
	return r.metaMap
}

func (r *Response) GetData() types.IoBuffer {
	return r.data
}
