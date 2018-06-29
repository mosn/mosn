package types

import (
	"reflect"
	"testing"
)

func TestInitSet(t *testing.T) {
	type args struct {
		input []string
	}
	tests := []struct {
		name string
		args args
		want SortedStringSetType
	}{
		{
			name: "testcase1",
			args: args{
				input: []string{"ac", "ab", "cd"},
			},
			want: SortedStringSetType{
				keys: []string{
					"ab", "ac", "cd",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InitSet(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitSortedMap(t *testing.T) {
	type args struct {
		input map[string]string
	}
	
	inmap := map[string]string{
		"hello":"yes",
		"bb":"yes",
		"aa":"no",
	}
	
	tests := []struct {
		name string
		args args
		want SortedMap
	}{
		{
			name:"case1",
			args:args{
				input:inmap,
			},
			want:SortedMap{
				Content:map[string]string{
					"aa":"no",
					"bb":"yes",
					"hello":"yes",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := InitSortedMap(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitSortedMap() = %v, want %v", got, tt.want)
			}
		})
	}
}
