package bolt

import (
	"context"
	"github.com/stretchr/testify/assert"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
	"testing"
)

const content = "this is the content"

func getEncodedReqBuf() types.IoBuffer {
	// request build
	// step 1, build a original request
	var req = NewRpcRequest(123454321, protocol.CommonHeader{
		"k1": "v1",
		"k2": "v2",
	}, buffer.NewIoBufferString(content))

	// step 2, encode this request
	encodedBuf, _ := encodeRequest(context.Background(), req)
	return encodedBuf
}

// the header set behaviour should not override the content of user
// https://github.com/mosn/mosn/issues/1393
func TestHeaderOverrideBody(t *testing.T) {
	buf := getEncodedReqBuf()

	// step 1, decode the bytes
	cmd, _ := decodeRequest(context.Background(), buf, false)

	// step 2, set the header of the decoded request
	req := cmd.(*Request)
	req.Header.Set("k2", "longer_header_value")

	// step 3, re-encode again
	buf, err := encodeRequest(context.Background(), req)
	assert.Nil(t, err)

	// step 4, re-decode
	cmdNew, err := decodeRequest(context.Background(), buf, false)
	assert.Nil(t, err)

	// step 5, the final decoded content should be equal to the original
	reqNew := cmdNew.(*Request)
	assert.Equal(t, reqNew.Content.String(), content)
}
