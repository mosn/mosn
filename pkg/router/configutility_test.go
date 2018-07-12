package router
//
//import (
//	"testing"
//
//	"github.com/alipay/sofamosn/pkg/types"
//)
//
//func TestConfigUtility_MatchQueryParams(t *testing.T) {
//	type fields struct {
//		HeaderData            types.HeaderData
//		QueryParameterMatcher QueryParameterMatcher
//	}
//	type args struct {
//		queryParams       types.QueryParams
//		configQueryParams []types.QueryParameterMatcher
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   bool
//	}{
//		{
//
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			cu := &ConfigUtility{
//				HeaderData:            tt.fields.HeaderData,
//				QueryParameterMatcher: tt.fields.QueryParameterMatcher,
//			}
//			if got := cu.MatchQueryParams(tt.args.queryParams, tt.args.configQueryParams); got != tt.want {
//				t.Errorf("ConfigUtility.MatchQueryParams() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
