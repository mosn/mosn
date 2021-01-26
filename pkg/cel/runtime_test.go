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

package cel

import (
	"context"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"testing"
	"time"

	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/cel/ext"
	"mosn.io/mosn/pkg/protocol"
)

func TestMosnCtx(t *testing.T) {
	compiler := NewExpressionBuilder(map[string]attribute.Kind{
		"ctx": attribute.MOSN_CTX,
	}, CompatCEXL)
	expression, typ, err := compiler.Compile(`ctx.rewrite_request_url("xx")`)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(typ)
	bag := attribute.NewMutableBag(nil)

	bag.Set("ctx", context.Background())
	out, err := expression.Evaluate(bag)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(out)
}

func TestSample(t *testing.T) {
	compiler := NewExpressionBuilder(map[string]attribute.Kind{
		"source.header": attribute.STRING_MAP,
		"a.ip":          attribute.IP_ADDRESS,
		"a.url":         attribute.URI,
	}, CompatCEXL)
	expression, typ, err := compiler.Compile(`int(source.header["a"]) > 10 && source.header["b"] == "hello"`)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(typ)
	bag := attribute.NewMutableBag(nil)

	bag.Set("source.header", protocol.CommonHeader(map[string]string{
		"a": "11",
		"b": "hello",
	}))
	out, err := expression.Evaluate(bag)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(out)

	expression, _, err = compiler.Compile(`string(a.ip) == "1.1.1.1" && string(a.url) == "http://127.0.0.1"`)
	if err != nil {
		t.Fatal(err)
	}

	u, _ := url.Parse("http://127.0.0.1")
	bag.Set("a.ip", net.IPv4(1, 1, 1, 1))
	bag.Set("a.url", u)

	_, err = expression.Evaluate(bag)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDuration(t *testing.T) {
	compiler := NewExpressionBuilder(map[string]attribute.Kind{
		"source.time": attribute.DURATION,
	}, CompatCEXL)
	expression, typ, err := compiler.Compile(`int(source.time | "0") / int("1ms")`)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(typ)
	bag := attribute.NewMutableBag(nil)

	bag.Set("source.time", time.Second)
	out, err := expression.Evaluate(bag)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(out)
}

func TestMoreCases(t *testing.T) {
	compiler := NewExpressionBuilder(map[string]attribute.Kind{
		"request.headers": attribute.STRING_MAP,
		"request.code":    attribute.INT64,
		"source.label":    attribute.STRING_MAP,
		"request.host":    attribute.STRING,
		"request.path":    attribute.STRING,
		"source.ip":       attribute.IP_ADDRESS,
		"other.duration":  attribute.DURATION,
		"other.double":    attribute.DOUBLE,
	}, CompatCEXL)
	bag := attribute.NewMutableBagForMap(map[string]interface{}{
		"request.headers": protocol.CommonHeader(map[string]string{
			"times": "10",
		}),
		"request.code": 200,
		"source.label": protocol.CommonHeader(map[string]string{
			"app": "test_app",
			"svc": "test_svc",
		}),
		"request.host":   "test.local",
		"request.path":   "/url",
		"source.ip":      []byte(net.IPv4(10, 10, 10, 10)),
		"other.duration": time.Second,
		"other.double":   1.1,
	})

	type args struct {
		src string
		bag attribute.Bag
	}
	tests := []struct {
		name            string
		args            args
		want            interface{}
		wantKind        attribute.Kind
		wantCompileErr  bool
		wantEvaluateErr bool
	}{
		{
			args: args{
				src: `true`,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `false`,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `200`,
			},
			want:     int64(200),
			wantKind: attribute.INT64,
		},
		{
			args: args{
				src: `"hello world"`,
			},
			want:     "hello world",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `request.code`,
			},
			wantKind:        attribute.INT64,
			wantEvaluateErr: true,
		},
		{
			args: args{
				src: `request.code`,
				bag: bag,
			},
			want:     int64(200),
			wantKind: attribute.INT64,
		},
		{
			args: args{
				src: `request.code == 200`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code == int("200")`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code == "200"`,
				bag: bag,
			},
			wantKind:       attribute.VALUE_TYPE_UNSPECIFIED,
			wantCompileErr: true,
		},
		{
			args: args{
				src: `request.code == 400`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code != 200`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code != 400`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code > 200`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code < 200`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code >= 200`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code <= 200`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code != 200 || source.label["app"] == "test_app"`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code == 200 && source.label["app"] != "test_app"`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.code == 200 && source.label["app"] == "test_app"`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.host + request.path`,
				bag: bag,
			},
			want:     "test.local/url",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `"hello".reverse()`,
				bag: bag,
			},
			want:     "olleh",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `request.headers["t"] | source.label["t"]`,
				bag: bag,
			},
			wantEvaluateErr: true,
			wantKind:        attribute.STRING,
		},
		{
			args: args{
				src: `request.headers["t"] | source.label["t"] | "unknown"`,
				bag: bag,
			},
			want:     "unknown",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `request.headers["svc"] | source.label["svc"] | "unknown"`,
				bag: bag,
			},
			want:     "test_svc",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `request.host`,
				bag: bag,
			},
			want:     "test.local",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `match(request.host, "*.local")`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `match(request.host, "test.*")`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `match(request.host, "test.local")`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `match(request.host, "test*local")`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.host.startsWith("test.")`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.host.endsWith(".local")`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.host.startsWith("test.xxx")`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `request.host.endsWith("xxx.local")`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `"^test".matches(request.host)`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `"local$".matches(request.host)`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `"test|local".matches(request.host)`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `emptyStringMap()`,
				bag: bag,
			},
			want:     protocol.CommonHeader(map[string]string{}),
			wantKind: attribute.STRING_MAP,
		},
		{
			args: args{
				src: `source.label | emptyStringMap()`,
				bag: bag,
			},
			want: protocol.CommonHeader(map[string]string{
				"app": "test_app",
				"svc": "test_svc",
			}),
			wantKind: attribute.STRING_MAP,
		},
		{
			args: args{
				src: `conditional((request.path | "/") == "/url", "200", "404")`,
				bag: bag,
			},
			want:     "200",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `conditional((request.path | "/") == "/url2", "200", "404")`,
				bag: bag,
			},
			want:     "404",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `conditionalString((request.path | "/") == "/url", "200", "404")`,
				bag: bag,
			},
			want:     "200",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `conditionalString((request.path | "/") == "/url2", "200", "404")`,
				bag: bag,
			},
			want:     "404",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `size(request.path)`,
				bag: bag,
			},
			want:     int64(4),
			wantKind: attribute.INT64,
		},
		{
			args: args{
				src: `toLower("User-Agent")`,
				bag: bag,
			},
			want:     "user-agent",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `toUpper("User-Agent")`,
				bag: bag,
			},
			want:     "USER-AGENT",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `ip("1.1.1.1")`,
				bag: bag,
			},
			want:     net.IPv4(1, 1, 1, 1),
			wantKind: attribute.IP_ADDRESS,
		},
		{
			args: args{
				src: `source.ip == ip("1.1.1.1")`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `timestamp("2015-01-02T15:04:05Z")`,
				bag: bag,
			},
			want:     time.Date(2015, 1, 2, 15, 4, 5, 0, time.UTC),
			wantKind: attribute.TIMESTAMP,
		},
		{
			args: args{
				src: `uri("https://test.local")`,
				bag: bag,
			},
			want:     &url.URL{Scheme: "https", Host: "test.local"},
			wantKind: attribute.URI,
		},
		{
			args: args{
				src: `uri("https://test.local") == uri("https://test.local")`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `uri("https://test.local") == uri("https://test2.local")`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `email("test@test.local")`,
				bag: bag,
			},
			want:     &mail.Address{Address: "test@test.local"},
			wantKind: attribute.EMAIL_ADDRESS,
		},
		{
			args: args{
				src: `email("test@test.local") == email("test@test.local")`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `email("test@test.local") == email("test@test2.local")`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `dnsName("test.local")`,
				bag: bag,
			},
			want:     &ext.DNSName{Name: "test.local"},
			wantKind: attribute.DNS_NAME,
		},
		{
			args: args{
				src: `dnsName(request.host) == dnsName("test.local")`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `dnsName(request.host) == dnsName("test2.local")`,
				bag: bag,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `duration("1s") == other.duration`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `"1s" == other.duration`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `other.double == 1.1`,
				bag: bag,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `bool("true")`,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `bool("false")`,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `bool("T")`,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `bool("F")`,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `bool("TRUE")`,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `bool("FALSE")`,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `bool("t")`,
			},
			want:     true,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `bool("f")`,
			},
			want:     false,
			wantKind: attribute.BOOL,
		},
		{
			args: args{
				src: `bool("x")`,
			},
			wantEvaluateErr: true,
			wantKind:        attribute.BOOL,
		},
		{
			args: args{
				src: `string(true)`,
			},
			want:     "true",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `string(false)`,
			},
			want:     "false",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `string(10)`,
			},
			want:     "10",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `string(10.0)`,
			},
			want:     "10",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `string(10.1)`,
			},
			want:     "10.1",
			wantKind: attribute.STRING,
		},
		{
			args: args{
				src: `int(true)`,
			},
			wantCompileErr: true,
			wantKind:       attribute.VALUE_TYPE_UNSPECIFIED,
		},
		{
			args: args{
				src: `int(false)`,
			},
			wantCompileErr: true,
			wantKind:       attribute.VALUE_TYPE_UNSPECIFIED,
		},
		{
			args: args{
				src: `int("10")`,
			},
			want:     int64(10),
			wantKind: attribute.INT64,
		},
		{
			args: args{
				src: `int("10.0")`,
			},
			wantEvaluateErr: true,
			wantKind:        attribute.INT64,
		},
		{
			args: args{
				src: `double(true)`,
			},
			wantCompileErr: true,
			wantKind:       attribute.VALUE_TYPE_UNSPECIFIED,
		},
		{
			args: args{
				src: `double(false)`,
			},
			wantCompileErr: true,
			wantKind:       attribute.VALUE_TYPE_UNSPECIFIED,
		},
		{
			args: args{
				src: `double("10")`,
			},
			want:     10.,
			wantKind: attribute.DOUBLE,
		},
		{
			args: args{
				src: `double("10.0")`,
			},
			want:     10.,
			wantKind: attribute.DOUBLE,
		},
		{
			args: args{
				src: `double("10.1")`,
			},
			want:     10.1,
			wantKind: attribute.DOUBLE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.args.src, func(t *testing.T) {
			expression, kind, err := compiler.Compile(tt.args.src)
			if (err != nil) != tt.wantCompileErr {
				t.Errorf("Compile() error = %v, wantCompileErr %v", err, tt.wantCompileErr)
				return
			}
			if kind != tt.wantKind {
				t.Errorf("Compile() kind, got = %v, want %v", kind, tt.wantKind)
				return
			}
			if kind == attribute.VALUE_TYPE_UNSPECIFIED {
				return
			}

			got, err := expression.Evaluate(tt.args.bag)
			if (err != nil) != tt.wantEvaluateErr {
				t.Errorf("Evaluate() error = %v, wantEvaluateErr %v", err, tt.wantEvaluateErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Evaluate() got = %#v, want %v", got, tt.want)
			}
		})
	}
}
