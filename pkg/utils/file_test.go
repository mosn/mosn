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
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"testing"
)

func TestWriteFileSafety(t *testing.T) {
	target := "/tmp/test_write_file_safety"
	data := []byte("test_data")
	if err := WriteFileSafety(target, data, 0644); err != nil {
		t.Fatal("write file error: ", err)
	}
	// verify
	b, err := ioutil.ReadFile(target)
	if err != nil {
		t.Fatal("read target file failed: ", err)
	}
	if !bytes.Equal(data, b) {
		t.Error("write data is not expected")
	}

	f, err := os.Stat(target)
	if err != nil {
		t.Fatal("read target file stat failed: ", err)
	}
	if !(f.Mode() == 0644) {
		t.Fatal("target file stat verify failed: ", f.Mode())
	}
}

func TestReadJsonFile(t *testing.T) {
	AnyError := errors.New("any error")
	tcs := []struct {
		path     string
		data     []byte
		expected error
	}{
		{
			path:     "/tmp/test_empty.json",
			data:     []byte(`{"test":"data"}`),
			expected: nil,
		},
		{
			path:     "/tmp/not.json.tmp",
			data:     []byte("12345"),
			expected: ErrIgnore,
		},
		{
			path:     "/tmp/empty.json",
			data:     nil,
			expected: ErrIgnore,
		},
		{
			path:     "/tmp/error.json",
			data:     []byte("12345"),
			expected: AnyError,
		},
	}
	for _, tc := range tcs {
		os.Remove(tc.path)
		if err := WriteFileSafety(tc.path, tc.data, 0644); err != nil {
			t.Fatal(err)
		}
		v := map[string]interface{}{}
		err := ReadJsonFile(tc.path, &v)
		switch tc.expected {
		case ErrIgnore, nil:
			if err != tc.expected {
				t.Fatalf("expected %v, but got: %v", tc.expected, err)
			}
		case AnyError:
			if err == nil {
				t.Fatal("expected got an error, but not")
			}
		}
	}
}
