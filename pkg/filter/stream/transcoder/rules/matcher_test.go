package rules

import (
	"context"
	"github.com/valyala/fasthttp"
	"mosn.io/mosn/pkg/protocol/http"
	"reflect"
	"testing"

	"mosn.io/mosn/pkg/types"
)

func TestTransferRuleConfigMatches(t *testing.T) {
	type fields struct {
		MatcherConfig *MatcherConfig
		RuleInfo      *RuleInfo
		MatchType     string
	}
	type args struct {
		ctx     context.Context
		headers types.HeaderMap
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *RuleInfo
		want1  bool
	}{
		{
			name: "TestTransferRuleConfigMatches_match",
			fields: fields{
				MatcherConfig: &MatcherConfig{
					Headers: []HeaderMatcher{
						{
							Name:  "serviceCode",
							Value: "dsr",
						},
					},
				},
				RuleInfo: &RuleInfo{
					UpstreamProtocol: "a",
				},
				MatchType: "simpleMatcher",
			},
			args: args{
				ctx:     context.Background(),
				headers: buildHttpRequestHeaders(map[string]string{"serviceCode": "dsr"}),
			},
			want: &RuleInfo{
				UpstreamProtocol: "a",
			},
			want1: true,
		},
		{
			name: "TestTransferRuleConfigMatches_no_match",
			fields: fields{
				MatcherConfig: &MatcherConfig{
					Headers: []HeaderMatcher{
						{
							Name:  "serviceCode",
							Value: "dsr",
						},
					},
				},
				RuleInfo: &RuleInfo{
					UpstreamProtocol: "a",
				},
				MatchType: "simpleMatcher",
			},
			args: args{
				ctx:     context.Background(),
				headers: buildHttpRequestHeaders(map[string]string{"serviceCode": "ooo"}),
			},
			want:  nil,
			want1: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := &TransferRuleConfig{
				MatcherConfig: tt.fields.MatcherConfig,
				RuleInfo:      tt.fields.RuleInfo,
			}
			got, got1 := tf.Matches(tt.args.ctx, tt.args.headers, tt.fields.MatchType)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Matches() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Matches() got1 = %v, want %v", got1, tt.want1)
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
