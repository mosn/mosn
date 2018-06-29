package types

import (
	"reflect"
	"testing"
	"crypto/md5"
	"fmt"
)

func TestEqualHashValue(t *testing.T) {
	
	data := []byte("hello go")
	h := md5.Sum(data)
	
	fmt.Println(h)
	
	type args struct {
		h1 HashedValue
		h2 HashedValue
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
	 
		{
			name:"test1",
			args:args{
				h1:HashedValue{143,93,73,242,226, 204, 175, 92, 16, 68, 2, 109, 20, 181, 104, 132},
				h2:HashedValue{143,93,73,242,226, 204, 175, 92, 16, 68, 2, 109, 20, 181, 104, 132},
				
			},
			want:true,
		},
		{
			name:"test2",
			args:args{
				h1:HashedValue{143,93,73,242,226, 204, 175, 92, 16, 68, 2, 109, 20, 181, 104, 132},
				h2:HashedValue{143,93,73,242,226, 204, 175, 92, 16, 68, 2, 109, 20, 181, 104, 133},
				
			},
			want:false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EqualHashValue(tt.args.h1, tt.args.h2); got != tt.want {
				t.Errorf("EqualHashValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateHashedValue(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want HashedValue
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GenerateHashedValue(tt.args.input); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GenerateHashedValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
