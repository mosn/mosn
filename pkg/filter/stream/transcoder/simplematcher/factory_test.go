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

package simplematcher

import (
	"context"
	"github.com/valyala/fasthttp"
	"mosn.io/mosn/pkg/filter/stream/transcoder/rules"
	"mosn.io/mosn/pkg/protocol/http"
	"reflect"
	"testing"

	"mosn.io/mosn/pkg/types"
)

func TestRuleMatches(t *testing.T) {
	type fields struct {
		Macther rules.RuleMatcher
	}
	type args struct {
		ctx     context.Context
		headers types.HeaderMap
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   rules.RuleMatcher
		want1  bool
	}{
		{
			name: "TestRuleMatches_match",
			fields: fields{
				Macther: rules.NewMatcher(&rules.MatcherConfig{
					MatcherType: "SimpleMatcher",
				}),
			},
			args: args{
				ctx:     context.Background(),
				headers: buildHttpRequestHeaders(map[string]string{"serviceCode": "dsr"}),
			},
			want:  &SimpleRuleMatcher{},
			want1: true,
		},
		{
			name: "TestTRuleMatches_no_match",
			fields: fields{
				Macther: rules.NewMatcher(&rules.MatcherConfig{
					MatcherType: "SimpleMatcher2",
				}),
			},
			args: args{
				ctx:     context.Background(),
				headers: buildHttpRequestHeaders(map[string]string{"serviceCode": "ooo"}),
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fields.Macther
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Matches() got = %v, want %v", tt.fields.Macther, tt.want)
			}
			if tt.fields.Macther != nil {
				got1 := tt.fields.Macther.Matches(tt.args.ctx, tt.args.headers)
				if !reflect.DeepEqual(got1, tt.want1) {
					t.Errorf("Matches() got = %v, want %v", got1, tt.want1)
				}
			}
		})
	}
}

func TestDefaultMatches(t *testing.T) {
	type args struct {
		ctx     context.Context
		headers types.HeaderMap
	}
	tests := []struct {
		name  string
		rules []*rules.TransferRule
		args  args
		want  *rules.RuleInfo
		want1 bool
	}{
		{
			name: "TestRuleMatches_match",
			rules: []*rules.TransferRule{{
				Macther: rules.NewMatcher(&rules.MatcherConfig{
					MatcherType: "SimpleMatcher",
				}),
				RuleInfo: &rules.RuleInfo{
					UpstreamProtocol: "a",
				},
			},
			},
			args: args{
				ctx:     context.Background(),
				headers: buildHttpRequestHeaders(map[string]string{"serviceCode": "dsr"}),
			},
			want: &rules.RuleInfo{
				UpstreamProtocol: "a",
			},
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := rules.DefaultMatches(tt.args.ctx, tt.args.headers, tt.rules)
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Matches() got = %v, want %v", got1, tt.want1)
			}
			if got.UpstreamProtocol != tt.want.UpstreamProtocol {
				t.Errorf("Matches() got = %v, want %v", got.UpstreamProtocol, tt.want.UpstreamProtocol)
			}
		})
	}
}

func buildHttpRequestHeaders(args map[string]string) http.RequestHeader {
	header := &fasthttp.RequestHeader{}

	for key, value := range args {
		header.Set(key, value)
	}

	return http.RequestHeader{RequestHeader: header}
}
