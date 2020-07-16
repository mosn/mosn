package ext

import (
	"testing"

	"github.com/google/cel-go/interpreter/functions"
)

func Test_reflectWrapFuncUnaryOp(t *testing.T) {
	type args struct {
		fun interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args: args{
				fun: func(a int64) int64 {
					return 0
				},
			},
		},
		{
			args: args{
				fun: func(a int64) (int64, error) {
					return 0, nil
				},
			},
		},
		{
			args: args{
				fun: func(a int64) (int64, int64, error) {
					return 0, 0, nil
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reflectWrapFunc(tt.args.fun)
			if (err != nil) != tt.wantErr {
				t.Errorf("reflectWrapFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				if _, ok := got.(functions.UnaryOp); !ok {
					t.Errorf("reflectWrapFunc() got = %v", got)
				}
			}
		})
	}
}

func Test_reflectWrapFuncBinaryOp(t *testing.T) {
	type args struct {
		fun interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args: args{
				fun: func(a, b int64) int64 {
					return 0
				},
			},
		},
		{
			args: args{
				fun: func(a, b int64) (int64, error) {
					return 0, nil
				},
			},
		},
		{
			args: args{
				fun: func(a, b int64) (int64, int64, error) {
					return 0, 0, nil
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reflectWrapFunc(tt.args.fun)
			if (err != nil) != tt.wantErr {
				t.Errorf("reflectWrapFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				if _, ok := got.(functions.BinaryOp); !ok {
					t.Errorf("reflectWrapFunc() got = %v", got)
				}
			}
		})
	}
}

func Test_reflectWrapFuncFunctionOp(t *testing.T) {
	type args struct {
		fun interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			args: args{
				fun: func() int64 {
					return 0
				},
			},
		},
		{
			args: args{
				fun: func() (int64, error) {
					return 0, nil
				},
			},
		},
		{
			args: args{
				fun: func() (int64, int64, error) {
					return 0, 0, nil
				},
			},
			wantErr: true,
		},
		{
			args: args{
				fun: func(a, b, c int64) int64 {
					return 0
				},
			},
		},
		{
			args: args{
				fun: func(a, b, c int64) (int64, error) {
					return 0, nil
				},
			},
		},
		{
			args: args{
				fun: func(a, b, c int64) (int64, int64, error) {
					return 0, 0, nil
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := reflectWrapFunc(tt.args.fun)
			if (err != nil) != tt.wantErr {
				t.Errorf("reflectWrapFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil {
				if _, ok := got.(functions.FunctionOp); !ok {
					t.Errorf("reflectWrapFunc() got = %v", got)
				}
			}
		})
	}
}
