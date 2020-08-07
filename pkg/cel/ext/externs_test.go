/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package ext

import (
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"testing"
)

func Test_externDNSName(t *testing.T) {
	type args struct {
		in string
	}
	tests := []struct {
		name    string
		args    args
		want    *DNSName
		wantErr bool
	}{
		{
			args: args{
				in: "test.local",
			},
			want: &DNSName{"test.local"},
		},
		{
			args: args{
				in: "test..local",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := externDNSName(tt.args.in)
			if (err != nil) != tt.wantErr {
				t.Errorf("externDNSName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("externDNSName() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getEmailParts(t *testing.T) {
	type args struct {
		email string
	}
	tests := []struct {
		name       string
		args       args
		wantLocal  string
		wantDomain string
	}{
		{
			args: args{
				email: "test@test.local",
			},
			wantLocal:  "test",
			wantDomain: "test.local",
		},
		{
			args: args{
				email: "test.local",
			},
			wantLocal:  "test.local",
			wantDomain: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLocal, gotDomain := getEmailParts(tt.args.email)
			if gotLocal != tt.wantLocal {
				t.Errorf("getEmailParts() gotLocal = %v, want %v", gotLocal, tt.wantLocal)
			}
			if gotDomain != tt.wantDomain {
				t.Errorf("getEmailParts() gotDomain = %v, want %v", gotDomain, tt.wantDomain)
			}
		})
	}
}

func Test_externMatch(t *testing.T) {
	type args struct {
		str     string
		pattern string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			args: args{
				str:     "test.local",
				pattern: "test.*",
			},
			want: true,
		},
		{
			args: args{
				str:     "test.local",
				pattern: "*.local",
			},
			want: true,
		},
		{
			args: args{
				str:     "test.local",
				pattern: "test.local",
			},
			want: true,
		},
		{
			args: args{
				str:     "test.local",
				pattern: "test*local",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := externMatch(tt.args.str, tt.args.pattern); got != tt.want {
				t.Errorf("externMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_externReverse(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{
				s: "",
			},
			want: "",
		},
		{
			args: args{
				s: "hello",
			},
			want: "olleh",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := externReverse(tt.args.s); got != tt.want {
				t.Errorf("externReverse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_externConditionalString(t *testing.T) {
	type args struct {
		condition bool
		trueStr   string
		falseStr  string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{
				condition: true,
				trueStr:   "trueStr",
				falseStr:  "falseStr",
			},
			want: "trueStr",
		},
		{
			args: args{
				condition: false,
				trueStr:   "trueStr",
				falseStr:  "falseStr",
			},
			want: "falseStr",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := externConditionalString(tt.args.condition, tt.args.trueStr, tt.args.falseStr); got != tt.want {
				t.Errorf("externConditionalString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_externIPEqual(t *testing.T) {
	type args struct {
		a []byte
		b []byte
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			args: args{
				a: net.IPv4(1, 0, 0, 0),
				b: net.IPv4(1, 0, 0, 0),
			},
			want: true,
		},
		{
			args: args{
				a: net.IPv4(1, 0, 0, 0),
				b: net.IPv4(1, 0, 0, 1),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := externIPEqual(tt.args.a, tt.args.b); got != tt.want {
				t.Errorf("externIPEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_externDNSNameEqual(t *testing.T) {
	type args struct {
		n1 string
		n2 string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			args: args{
				n1: "test.local",
				n2: "test.local",
			},
			want: true,
		},
		{
			args: args{
				n1: "test.local",
				n2: "test2.local",
			},
			want: false,
		},
		{
			args: args{
				n1: "/local",
				n2: "test2.local",
			},
			want:    false,
			wantErr: true,
		},
		{
			args: args{
				n1: "test.local",
				n2: "/local",
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := externDNSNameEqual(tt.args.n1, tt.args.n2)
			if (err != nil) != tt.wantErr {
				t.Errorf("externDNSNameEqual() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("externDNSNameEqual() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_externEmailEqual(t *testing.T) {
	type args struct {
		e1 string
		e2 string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			args: args{
				e1: "test@test.local",
				e2: "test@test.local",
			},
			want: true,
		},
		{
			args: args{
				e1: "test@test.local",
				e2: "test@test2.local",
			},
			want: false,
		},
		{
			args: args{
				e1: "test.local",
				e2: "test@test.local",
			},
			want:    false,
			wantErr: true,
		},
		{
			args: args{
				e1: "test@test.local",
				e2: "test.local",
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				got    bool
				a1, a2 *mail.Address
				err    error
			)
			a1, err = mail.ParseAddress(tt.args.e1)
			if err == nil {
				a2, err = mail.ParseAddress(tt.args.e2)
				if err == nil {
					got, err = externEmailEqual(a1, a2)
				}
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("externEmailEqual() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("externEmailEqual() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_externURIEqual(t *testing.T) {
	type args struct {
		u1 string
		u2 string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			args: args{
				u1: "https://test.local/",
				u2: "https://test.local/",
			},
			want: true,
		},
		{
			args: args{
				u1: "https://test.local/",
				u2: "https://test.local:443/",
			},
			want: false,
		},
		{
			args: args{
				u1: "https://test.local/",
				u2: "https://test2.local/",
			},
			want: false,
		},
		{
			args: args{
				u1: "https://test.local/",
				u2: ":test.local/",
			},
			want:    false,
			wantErr: true,
		},
		{
			args: args{
				u1: ":test.local/",
				u2: "https://test.local/",
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var (
				got    bool
				a1, a2 *url.URL
				err    error
			)

			a1, err = url.Parse(tt.args.u1)
			if err == nil {
				a2, err = url.Parse(tt.args.u2)
				if err == nil {
					got, err = externURIEqual(a1, a2)
				}
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("externURIEqual() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("externURIEqual() got = %v, want %v", got, tt.want)
			}
		})
	}
}
