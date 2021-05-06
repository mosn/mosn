package tunnel

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteBuffer(t *testing.T) {
	c := &ConnectionInitInfo{
		ClusterName: "test_c1",
		Weight:      10,
		HostName:    "test100",
	}
	buffer, err := WriteBuffer(c)
	if err != nil {
		assert.Error(t, err, "write connection info failed")
		return
	}

	assert.NotEmpty(t, buffer.Bytes(), "buffer is empty")

	type Temp struct {
		name string
		id   int64
	}

	temp := &Temp{
		name: "aaa",
		id:   100,
	}

	_, err = WriteBuffer(temp)
	if err == nil {
		assert.Error(t, err, "expect to fail but succeed")
	}
}
func TestWriteAndRead(t *testing.T) {
	c := &ConnectionInitInfo{
		ClusterName: "test_c1",
		Weight:      10,
		HostName:    "test100",
	}
	buffer, err := WriteBuffer(c)
	if err != nil {
		assert.Error(t, err, "write connection info failed")
		return
	}

	res := Read(buffer)
	assert.EqualValues(t, res, c, "different between writes and reads")
}
