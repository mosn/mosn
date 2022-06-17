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

package trace

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
)

func Test_pluginDriver_Get(t *testing.T) {
	type fields struct {
		tracer api.Tracer
	}
	type args struct {
		proto types.ProtocolName
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   api.Tracer
	}{
		{
			name: "test",
			fields: fields{
				tracer: api.Tracer(nil),
			},
			args: args{},
			want: api.Tracer(nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &pluginDriver{
				tracer: tt.fields.tracer,
			}
			assert.Equalf(t, tt.want, d.Get(tt.args.proto), "Get(%v)", tt.args.proto)
		})
	}
}

func Test_pluginDriver_Init(t *testing.T) {
	type args struct {
		config map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "path is empty",
			args: args{
				config: map[string]interface{}{
					"sopath": "",
				},
			},
			wantErr: assert.Error,
		},
		{
			name: "path is empty",
			args: args{
				config: map[string]interface{}{
					"sopath": "tracer.so",
				},
			},
			wantErr: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &pluginDriver{}
			tt.wantErr(t, d.Init(tt.args.config), fmt.Sprintf("Init(%v)", tt.args.config))
		})
	}
}

func Test_pluginDriver_Register(t *testing.T) {
	type fields struct {
		tracer api.Tracer
	}
	type args struct {
		proto   types.ProtocolName
		builder api.TracerBuilder
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &pluginDriver{
				tracer: tt.fields.tracer,
			}
			d.Register(tt.args.proto, tt.args.builder)
		})
	}
}
