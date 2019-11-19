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

package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestSetHijackStdPipeline(t *testing.T) {
	// init
	stderrFile := "/tmp/test_stderr"
	os.Remove(stderrFile)
	// call, test std error only
	SetHijackStdPipeline(stderrFile, false, true)
	time.Sleep(time.Second) // wait goroutine run
	fmt.Fprintf(os.Stderr, "test stderr")
	// verify
	if !verifyFile(stderrFile, "test stderr") {
		t.Error("stderr hijack failed")
	}
	ResetHjiackStdPipeline()
	fmt.Fprintf(os.Stderr, "repaired\n")
}

func verifyFile(p string, data string) bool {
	b, err := ioutil.ReadFile(p)
	if err != nil {
		return false
	}
	return string(b) == data
}
