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

package dubbo

import (
	"testing"
)

func TestMetadata_Register(t *testing.T) {
	type args struct {
		path    string
		nodes   []*Node
		version string // test expect version
	}
	tests := []struct {
		name string
		want bool
		args args
	}{
		{
			name: "register node #1",
			want: true,
			args: args{
				path: "com.alipay.sofa.Service666",
				nodes: []*Node{
					{
						Service: "com.alipay.sofa.Service",
						Version: "1.0",
					},
				},
				version: "1.0",
			},
		},
		{
			// test regiter any times.
			name: "register node #2",
			want: true,
			args: args{
				path: "com.alipay.sofa.Service666",
				nodes: []*Node{
					{
						Service: "com.alipay.sofa.Service",
						Version: "1.0",
					},
					{
						Service: "com.alipay.sofa.Service",
						Version: "1.0",
					},
				},
				version: "1.0",
			},
		},
		{
			name: "register node #3",
			want: true,
			args: args{
				path: "com.alipay.sofa.Service666",
				nodes: []*Node{
					{
						Service: "com.alipay.sofa.Service",
						Version: "1.0",
					},
					{
						Service: "com.alipay.sofa.Service",
						Version: "2.0",
					},
				},
				version: "1.0",
			},
		},
		{
			name: "register node #4",
			want: false,
			args: args{
				path: "com.alipay.sofa.Service666",
				nodes: []*Node{
					{
						Service: "com.alipay.sofa.Service",
						Version: "1.0",
					},
					{
						Service: "com.alipay.sofa.Service",
						Group:   "group",
					},
				},
				version: "group",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Metadata{}

			for _, node := range tt.args.nodes {
				m.Register(tt.args.path, node)
			}

			if _, matched := m.Find(tt.args.path, tt.args.version); matched != tt.want {
				t.Errorf("failed to find metadata by path %s and version %s, expected match %v, actual %v",
					tt.args.path, tt.args.version, tt.want, matched)
			}
		})
	}
}
