package dubbo

import (
	"encoding/binary"
	"github.com/alipay/sofa-mosn/pkg/types"
)

// dubbo command
type dubboCmd struct {
	data []byte

	requestID uint64
	header    map[string]string
}

func (d *dubboCmd) ProtocolCode() byte {
	// xprotocol unsupport protocol code ,so always return 0
	return 0
}

func (d *dubboCmd) RequestID() uint64 {
	if d.requestID == 0 {
		d.requestID = binary.BigEndian.Uint64(d.data[DUBBO_ID_IDX:(DUBBO_ID_IDX + DUBBO_ID_LEN)])
	}
	return d.requestID
}

func (d *dubboCmd) SetRequestID(requestID uint64) {
	binary.BigEndian.PutUint64(d.data[DUBBO_ID_IDX:], requestID)
}

// Header no use util we change multiplexing interface
func (d *dubboCmd) Header() map[string]string {
	return d.header
}

// Data no use util we change multiplexing interface
func (d *dubboCmd) Data() []byte {
	return d.data
}

// SetHeader no use util we change multiplexing interface
func (d *dubboCmd) SetHeader(header map[string]string) {
	d.header = header
}

// SetData no use util we change multiplexing interface
func (d *dubboCmd) SetData(data []byte) {
	if ok, _ := isValidDubboData(data); ok {
		d.data = data
	}

}

//Tracing
func (d *dubboCmd) GetServiceName(data []byte) string {
	if serviceNameFunc != nil {
		return serviceNameFunc(d.data)
	}
	return ""
}
func (d *dubboCmd) GetMethodName(data []byte) string {
	if methodNameFunc != nil {
		return methodNameFunc(d.data)
	}
	return ""
}

//RequestRouting
func (d *dubboCmd) GetMetas(data []byte) map[string]string {
	return nil
}

//ProtocolConvertor
func (d *dubboCmd) Convert(data []byte) (map[string]string, []byte) {
	return nil, nil
}

func (d *dubboCmd) Get(key string) (value string, ok bool) {
	value, ok = d.header[key]
	return
}

// Set key-value pair in header map, the previous pair will be replaced if exists
func (d *dubboCmd) Set(key string, value string) {
	d.header[key] = value
}

// Del delete pair of specified key
func (d *dubboCmd) Del(key string) {
	delete(d.header, key)
}

// Range calls f sequentially for each key and value present in the map.
// If f returns false, range stops the iteration.
func (d *dubboCmd) Range(f func(key, value string) bool) {
	for k, v := range d.header {
		// stop if f return false
		if !f(k, v) {
			break
		}
	}
}

// Clone used to deep copy header's map
func (d *dubboCmd) Clone() types.HeaderMap {
	return nil
}

// ByteSize return size of HeaderMap
func (d *dubboCmd) ByteSize() (size uint64) {
	for k, v := range d.header {
		size += uint64(len(k) + len(v))
	}
	return size
}
