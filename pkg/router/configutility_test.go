package router

import (
	"container/list"
	"reflect"
	"testing"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func TestConfigUtility_MatchHeaders(t *testing.T) {
	type args struct {
		requestHeaders map[string]string
		configHeaders  []*types.HeaderData
	}
	tests := []struct {
		name string
		cu   *ConfigUtility
		args args
		want bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cu.MatchHeaders(tt.args.requestHeaders, tt.args.configHeaders); got != tt.want {
				t.Errorf("ConfigUtility.MatchHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigUtility_MatchQueryParams(t *testing.T) {
	type args struct {
		queryParams       *types.QueryParams
		configQueryParams []types.QueryParameterMatcher
	}
	tests := []struct {
		name string
		cu   *ConfigUtility
		args args
		want bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cu.MatchQueryParams(tt.args.queryParams, tt.args.configQueryParams); got != tt.want {
				t.Errorf("ConfigUtility.MatchQueryParams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestQueryParameterMatcher_Matches(t *testing.T) {
	type args struct {
		requestQueryParams types.QueryParams
	}
	tests := []struct {
		name string
		qpm  *QueryParameterMatcher
		args args
		want bool
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.qpm.Matches(tt.args.requestQueryParams); got != tt.want {
				t.Errorf("QueryParameterMatcher.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigImpl_Name(t *testing.T) {
	tests := []struct {
		name string
		ci   *ConfigImpl
		want string
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ci.Name(); got != tt.want {
				t.Errorf("ConfigImpl.Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigImpl_Route(t *testing.T) {
	type args struct {
		headers     map[string]string
		randomValue uint64
	}
	tests := []struct {
		name string
		ci   *ConfigImpl
		args args
		want types.Route
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ci.Route(tt.args.headers, tt.args.randomValue); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConfigImpl.Route() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigImpl_InternalOnlyHeaders(t *testing.T) {
	tests := []struct {
		name string
		ci   *ConfigImpl
		want *list.List
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ci.InternalOnlyHeaders(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConfigImpl.InternalOnlyHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMetadataMatchCriteriaImpl_extractMetadataMatchCriteria(t *testing.T) {
	type fields struct {
		metadataMatchCriteria []types.MetadataMatchCriterion
	}
	type args struct {
		parent          *MetadataMatchCriteriaImpl
		metadataMatches map[string]interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mmcti := &MetadataMatchCriteriaImpl{
				metadataMatchCriteria: tt.fields.metadataMatchCriteria,
			}
			mmcti.extractMetadataMatchCriteria(tt.args.parent, tt.args.metadataMatches)
		})
	}
}
