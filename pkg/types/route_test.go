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
package types

import (
	"crypto/md5"
	"fmt"
	"reflect"
	"testing"
)

func TestEqualHashValue(t *testing.T) {

	data := []byte("hello go")
	h := md5.Sum(data)

	fmt.Println(h)

	type args struct {
		h1 HashedValue
		h2 HashedValue
	}
	tests := []struct {
		name string
		args args
		want bool
	}{

		{
			name: "test1",
			args: args{
				h1: HashedValue{143, 93, 73, 242, 226, 204, 175, 92, 16, 68, 2, 109, 20, 181, 104, 132},
				h2: HashedValue{143, 93, 73, 242, 226, 204, 175, 92, 16, 68, 2, 109, 20, 181, 104, 132},
			},
			want: true,
		},
		{
			name: "test2",
			args: args{
				h1: HashedValue{143, 93, 73, 242, 226, 204, 175, 92, 16, 68, 2, 109, 20, 181, 104, 132},
				h2: HashedValue{143, 93, 73, 242, 226, 204, 175, 92, 16, 68, 2, 109, 20, 181, 104, 133},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EqualHashValue(tt.args.h1, tt.args.h2); got != tt.want {
				t.Errorf("EqualHashValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateHashedValue(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want HashedValue
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateHashedValue(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateHashedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
