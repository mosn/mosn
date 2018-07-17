package http

import (
	"reflect"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/pkg/log"
)

func init(){
	log.InitDefaultLogger("",log.DEBUG)
}

func TestParseQueryString(t *testing.T) {
	type args struct {
		query string
	}
	tests := []struct {
		name string
		args args
		want types.QueryParams
	}{
		{
			args:args{
				query:"",
			},
			want:types.QueryParams{
			},
			
		},
		{
			args:args{
				query:"key1=valuex",
			},
			want:types.QueryParams{
				"key1":"valuex",
			},
			
		},
		
		{
			args:args{
				query:"key1=valuex&nobody=true",
			},
			want:types.QueryParams{
				"key1":"valuex",
				"nobody":"true",
			},
			
		},
		
		{
			args:args{
				query:"key1=valuex&nobody=true&test=biz",
			},
			want:types.QueryParams{
				"key1":"valuex",
				"nobody":"true",
				"test":"biz",
			},
			
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseQueryString(tt.args.query); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseQueryString() = %v, want %v", got, tt.want)
			}
		})
	}
}
