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

package skywalking

import (
	"context"
	"reflect"
	"testing"
	"time"

	v2 "mosn.io/mosn/pkg/config/v2"
)

func TestNewGO2SkyTracer(t *testing.T) {
	// use default config
	tracer, err := newGO2SkyTracer(nil)
	if err != nil {
		t.Errorf("new go2sky.Tracer error, err: %v", err)
	}
	s, _, err := tracer.CreateEntrySpan(context.Background(), "/NewGO2SkyTracer", func() (s string, err error) {
		return "", nil
	})
	if err != nil {
		t.Errorf("create entry span error, err: %v", err)
	}

	s.End()
	time.Sleep(time.Second)
}

func Test_parseAndVerifySkyTracerConfig(t *testing.T) {
	type args struct {
		cfg map[string]interface{}
	}
	tests := []struct {
		name       string
		args       args
		wantConfig v2.SkyWalkingTraceConfig
		wantErr    bool
	}{
		{
			name: "unknown reporter",
			args: args{
				cfg: map[string]interface{}{
					"reporter": "mosn",
				},
			},
			wantErr: true,
		},
		{
			name: "miss backend_service",
			args: args{
				cfg: map[string]interface{}{
					"reporter": v2.GRPCReporter,
				},
			},
			wantErr: true,
		},
		{
			name: "normal",
			args: args{
				cfg: map[string]interface{}{
					"reporter":        v2.GRPCReporter,
					"backend_service": "oap:11800",
					"service_name":    "normal",
					"with_register":   false,
				},
			},
			wantConfig: v2.SkyWalkingTraceConfig{
				Reporter:       v2.GRPCReporter,
				BackendService: "oap:11800",
				ServiceName:    "normal",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConfig, err := parseAndVerifySkyTracerConfig(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseAndVerifySkyTracerConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(gotConfig, tt.wantConfig) {
				t.Errorf("parseAndVerifySkyTracerConfig() gotConfig = %v, want %v", gotConfig, tt.wantConfig)
			}
		})
	}
}
