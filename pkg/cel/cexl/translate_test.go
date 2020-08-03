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

package cexl

import (
	"testing"
)

func TestSourceCEXLToCEL(t *testing.T) {
	type args struct {
		src string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			args: args{`"1s"`},
			want: `duration("1s")`,
		},
		{
			args: args{`aaa | bbb`},
			want: `pick(aaa, bbb)`,
		},
		{
			args: args{`aaa | bbb | ccc`},
			want: `pick(aaa, bbb, ccc)`,
		},
		{
			args: args{`aaa | bbb | ccc | ddd`},
			want: `pick(aaa, bbb, ccc, ddd)`,
		},
		{
			args: args{`aaa | pick(bbb, ccc)`},
			want: `pick(aaa, bbb, ccc)`,
		},
		{
			args: args{`aaa | "bbb"`},
			want: `pick(aaa, "bbb")`,
		},
		{
			args: args{`aaa | other(bbb)`},
			want: `pick(aaa, other(bbb))`,
		},
		{
			args: args{`other(aaa) | bbb`},
			want: `pick(other(aaa), bbb)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SourceCEXLToCEL(tt.args.src)
			if (err != nil) != tt.wantErr {
				t.Errorf("SourceCEXLToCEL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("SourceCEXLToCEL() got = %v, want %v", got, tt.want)
			}
		})
	}
}
