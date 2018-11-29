package example

import (
	"encoding/binary"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// example command
type exampleCmd struct {
	data []byte

	requestID uint64
	header    map[string]string
}

func (d *exampleCmd) ProtocolCode() byte {
	// xprotocol unsupport protocol code ,so always return 0
	return 0
}

func (d *exampleCmd) RequestID() uint64 {
	if d.requestID == 0 {
		d.requestID = binary.BigEndian.Uint64(d.data[ReqIDBeginOffset:(ReqIDBeginOffset + ReqIDLen)])
	}
	return d.requestID
}

func (d *exampleCmd) SetRequestID(requestID uint64) {
	binary.BigEndian.PutUint64(d.data[ReqIDBeginOffset:], requestID)
}

// Header no use util we change multiplexing interface
func (d *exampleCmd) Header() map[string]string {
	return d.header
}

// Data no use util we change multiplexing interface
func (d *exampleCmd) Data() []byte {
	return d.data
}

// SetHeader no use util we change multiplexing interface
func (d *exampleCmd) SetHeader(header map[string]string) {
	d.header = header
}

// SetData no use util we change multiplexing interface
func (d *exampleCmd) SetData(data []byte) {
	d.data = data
}

//Tracing
func (d *exampleCmd) GetServiceName(data []byte) string {
	return ""
}
func (d *exampleCmd) GetMethodName(data []byte) string {
	return ""
}

//RequestRouting
func (d *exampleCmd) GetMetas(data []byte) map[string]string {
	return nil
}

//ProtocolConvertor
func (d *exampleCmd) Convert(data []byte) (map[string]string, []byte) {
	return nil, nil
}

func (d *exampleCmd) Get(key string) (value string, ok bool) {
	value, ok = d.header[key]
	return
}

// Set key-value pair in header map, the previous pair will be replaced if exists
func (d *exampleCmd) Set(key string, value string) {
	d.header[key] = value
}

// Del delete pair of specified key
func (d *exampleCmd) Del(key string) {
	delete(d.header, key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (d *exampleCmd) Range(f func(key, value string) bool) {
	for k, v := range d.header {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
}

// Clone used to deep copy header's map
func (d *exampleCmd) Clone() types.HeaderMap {
	return nil
}

// ByteSize return size of HeaderMap
func (d *exampleCmd) ByteSize() (size uint64) {
	for k, v := range d.header {
		size += uint64(len(k) + len(v))
	}
	return size
}
