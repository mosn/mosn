/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package bolt

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/header"
)

const content = "this is the content"

func getEncodedReqBuf() api.IoBuffer {
	// request build
	// step 1, build a original request
	var req = NewRpcRequest(123454321, header.CommonHeader{
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
	req.BytesHeader.Set("k2", "longer_header_value")

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
