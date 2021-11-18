package rules

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTransferRuleManager(t *testing.T) {
	type fields struct {
		rules sync.Map
	}
	type args struct {
		listenerName string
		config       []*TransferRuleConfig
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestTransferRuleManager",
			fields: fields{},
			args: args{
				listenerName: "key",
				config: []*TransferRuleConfig{
					{
						MatcherConfig: &MatcherConfig{
							Headers: []HeaderMatcher{},
						},
						RuleInfo: &RuleInfo{
							UpstreamProtocol: "a",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trm := &TransferRuleManager{
				rules: tt.fields.rules,
			}
			if err := trm.AddOrUpdateTransferRule(tt.args.listenerName, tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("AddOrUpdateTransferRule() error = %v, wantErr %v", err, tt.wantErr)
			}

			trm.DeleteTransferRule(tt.args.listenerName)

			if _, ok := trm.GetTransferRule(tt.args.listenerName); ok {
				t.Errorf("GetTransferRule() ok = %v, want false", ok)
			}

			if err := trm.AddOrUpdateTransferRule(tt.args.listenerName, tt.args.config); (err != nil) != tt.wantErr {
				t.Errorf("AddOrUpdateTransferRule() error = %v, wantErr %v", err, tt.wantErr)
			}

			if conf, ok := trm.GetTransferRule(tt.args.listenerName); !ok {
				t.Errorf("GetTransferRule() ok = %v, want true", ok)
			} else {
				assert.Equal(t, conf, tt.args.config)
			}
		})
	}
}
