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
	"reflect"
	"testing"

	"github.com/valyala/fasthttp"
	"mosn.io/mosn/pkg/filter/stream/transcoder/matcher"
	"mosn.io/mosn/pkg/protocol/http"

	"mosn.io/mosn/pkg/types"
)

func TestSimpleMatcher(t *testing.T) {
	type fields struct {
		Matcher matcher.RuleMatcher
	}
	type args struct {
		ctx     context.Context
		headers types.HeaderMap
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   matcher.RuleMatcher
		want1  bool
	}{
		{
			name: "TestSimpleMatcher_match",
			fields: fields{
				Matcher: matcher.NewMatcher(&matcher.MatcherConfig{
					MatcherType: "simpleMatcher",
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
			name: "TestSimpleMatcher_no_match",
			fields: fields{
				Matcher: matcher.NewMatcher(&matcher.MatcherConfig{
					MatcherType: "simpleMatcher2",
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
			got := tt.fields.Matcher
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Matches() got = %v, want %v", tt.fields.Matcher, tt.want)
			}
			if tt.fields.Matcher != nil {
				got1 := tt.fields.Matcher.Matches(tt.args.ctx, tt.args.headers)
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
		rules []*matcher.TransferRule
		args  args
		want  *matcher.RuleInfo
		want1 bool
	}{
		{
			name: "TestDefaultMatches_no_header_match",
			rules: []*matcher.TransferRule{{
				Matcher: matcher.NewMatcher(&matcher.MatcherConfig{
					MatcherType: "simpleMatcher",
				}),
				RuleInfo: &matcher.RuleInfo{
					UpstreamProtocol: "a",
				},
			},
			},
			args: args{
				ctx:     context.Background(),
				headers: buildHttpRequestHeaders(map[string]string{"serviceCode": "dsr"}),
			},
			want: &matcher.RuleInfo{
				UpstreamProtocol: "a",
			},
			want1: true,
		},
		{
			name: "TestDefaultMatches_header_match",
			rules: []*matcher.TransferRule{{
				Matcher: matcher.NewMatcher(&matcher.MatcherConfig{
					MatcherType: "simpleMatcher",
					Config: map[string]interface{}{
						"name":  "serviceCode",
						"value": "dsr",
					},
				}),
				RuleInfo: &matcher.RuleInfo{
					UpstreamProtocol: "a",
				},
			},
			},
			args: args{
				ctx:     context.Background(),
				headers: buildHttpRequestHeaders(map[string]string{"serviceCode": "dsr"}),
			},
			want: &matcher.RuleInfo{
				UpstreamProtocol: "a",
			},
			want1: true,
		},
		{
			name: "TestDefaultMatches_header_no_match",
			rules: []*matcher.TransferRule{{
				Matcher: matcher.NewMatcher(&matcher.MatcherConfig{
					MatcherType: "simpleMatcher",
					Config: map[string]interface{}{
						"name":  "serviceCode",
						"value": "dsr2",
					},
				}),
				RuleInfo: &matcher.RuleInfo{
					UpstreamProtocol: "a",
				},
			},
			},
			args: args{
				ctx:     context.Background(),
				headers: buildHttpRequestHeaders(map[string]string{"serviceCode": "dsr"}),
			},
			want:  nil,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := matcher.DefaultMatches(tt.args.ctx, tt.args.headers, tt.rules)
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Matches() got = %v, want %v", got1, tt.want1)
			}
			if !reflect.DeepEqual(got, tt.want) {
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
