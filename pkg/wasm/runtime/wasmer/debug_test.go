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

package wasmer

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDebugParseDwarf(t *testing.T) {
	bytes, err := ioutil.ReadFile("./testdata/data.wasm")
	assert.Nil(t, err)

	debug := parseDwarf(bytes)
	assert.NotNil(t, debug)
	assert.NotNil(t, debug.data)
	assert.Equal(t, debug.codeSectionOffset, 0x326) // code section start addr

	lr := debug.getLineReader()
	assert.NotNil(t, lr)

	line := debug.SeekPC(uint64(0x2ef1)) // f3
	assert.NotNil(t, line)
}
