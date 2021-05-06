package tunnel

import (
	"reflect"
	"testing"

	"mosn.io/mosn/pkg/types"
)

func TestNewConnection(t *testing.T) {
	type args struct {
		config   ConnectionConfig
		listener types.Listener
	}
	tests := []struct {
		name string
		args args
		want *AgentRawConnection
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewConnection(tt.args.config, tt.args.listener); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewConnection() = %v, want %v", got, tt.want)
			}
		})
	}
}