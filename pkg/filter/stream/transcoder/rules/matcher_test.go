package rules

import (
	"context"
	"reflect"
	"testing"

	"mosn.io/mosn/pkg/types"
)

func TestTransferRuleConfigMatches(t *testing.T) {
	type fields struct {
		MatchType *MatcherConfig
		RuleInfo  *RuleInfo
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
			name: "TestTransferRuleConfigMatches",
			fields: fields{
				MatchType: &MatcherConfig{
					Headers: []HeaderMatcher{},
				},
				RuleInfo: &RuleInfo{
					UpstreamProtocol: "a",
				},
			},
			args: args{
				ctx:     context.Background(),
				headers: nil,
			},
			want: &RuleInfo{
				UpstreamProtocol: "a",
			},
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tf := &TransferRuleConfig{
				MatcherConfig: tt.fields.MatchType,
				RuleInfo:      tt.fields.RuleInfo,
			}
			got, got1 := tf.Matches(tt.args.ctx, tt.args.headers)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Matches() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Matches() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
