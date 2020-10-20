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

package stats

import (
	"reflect"
	"testing"
	"time"

	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/protocol"
)

func TestMetric(t *testing.T) {
	type args struct {
		conf       []*MetricConfig
		definition []*MetricDefinition
		bag        attribute.Bag
	}
	tests := []struct {
		name        string
		args        args
		want        []*Stat
		wantErr     bool
		wantStatErr bool
	}{
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name: "requests_total",
						Dimensions: map[string]string{
							"k1": `"v1"`,
						},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "requests_total",
						Value: `request.total_size | 0`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{
					"request.total_size": int64(100),
				}),
			},
			want: []*Stat{{Name: "requests_total", Labels: map[string]string{"k1": "v1"}, Value: 100}},
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name:  "requests_total",
						Match: `response.duration == "1s"`,
						Dimensions: map[string]string{
							"k1": `"v1"`,
						},
					},
					{
						Name:  "requests_total",
						Match: `response.duration == "2s"`,
						Dimensions: map[string]string{
							"k2": `"v2"`,
						},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "requests_total",
						Value: `request.total_size | 0`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{
					"request.total_size": int64(100),
					"response.duration":  time.Second,
				}),
			},
			want: []*Stat{{Name: "requests_total", Labels: map[string]string{"k1": "v1"}, Value: 100}},
		},

		{
			args: args{
				conf: []*MetricConfig{
					{
						Name: "requests_total",
						Dimensions: map[string]string{
							"k1": `"v1"`,
							"k2": `"v2"`,
						},
					},
					{
						Name:         "",
						TagsToRemove: []string{"k2"},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "requests_total",
						Value: `request.total_size | 0`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{
					"request.total_size": int64(100),
				}),
			},
			want: []*Stat{{Name: "requests_total", Labels: map[string]string{"k1": "v1"}, Value: 100}},
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name: "",
						Dimensions: map[string]string{
							"k1": `"v1"`,
						},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "request_duration_milliseconds",
						Value: `response.duration | "0"`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{
					"response.duration": time.Second,
				}),
			},
			want: []*Stat{{Name: "request_duration_milliseconds", Labels: map[string]string{"k1": "v1"}, Value: 1000}},
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name: "request_header",
						Dimensions: map[string]string{
							"header": `request.headers | emptyStringMap()`,
						},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "request_header",
						Value: `1`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{
					"request.headers": protocol.CommonHeader{"k1": "v1", "k2": "v2"},
				}),
			},
			want: []*Stat{{Name: "request_header", Labels: map[string]string{"header": "k1 => v1, k2 => v2"}, Value: 1}},
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name: "",
						Dimensions: map[string]string{
							"k1": `"v1"`,
						},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "request_duration_milliseconds",
						Value: `response.duration | "0"`,
					},
					{
						Name:  "requests_total",
						Value: `request.total_size | 0`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{
					"response.duration":  time.Second,
					"request.total_size": int64(100),
				}),
			},
			want: []*Stat{{Name: "request_duration_milliseconds", Labels: map[string]string{"k1": "v1"}, Value: 1000}, {Name: "requests_total", Labels: map[string]string{"k1": "v1"}, Value: 100}},
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name: "",
						Dimensions: map[string]string{
							"k1": `"v1"`,
						},
					},
					{
						Name: "requests_total",
						Dimensions: map[string]string{
							"k2": `"v2"`,
						},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "request_duration_milliseconds",
						Value: `response.duration | "0"`,
					},
					{
						Name:  "requests_total",
						Value: `request.total_size | 0`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{
					"response.duration":  time.Second,
					"request.total_size": int64(100),
				}),
			},
			want: []*Stat{{Name: "request_duration_milliseconds", Labels: map[string]string{"k1": "v1"}, Value: 1000}, {Name: "requests_total", Labels: map[string]string{"k1": "v1", "k2": "v2"}, Value: 100}},
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name: "want_error",
						Dimensions: map[string]string{
							"k1": `"v1"`,
						},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "want_error",
						Value: `"hello"`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{}),
			},
			wantErr: true,
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name:       "test",
						Dimensions: map[string]string{},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "test",
						Value: `response.duration | "0"`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{
					"response.duration": int64(time.Second),
				}),
			},
			wantStatErr: true,
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name: "test",
						Dimensions: map[string]string{
							"response_duration": `response.duration | "0"`,
						},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "test",
						Value: `200`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{
					"response.duration": int64(time.Second),
				}),
			},
			wantStatErr: true,
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name: "test",
						Dimensions: map[string]string{
							"response_code": `response.code | 200`,
						},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "test",
						Value: `200`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{
					"response.code": int64(404),
				}),
			},
			want: []*Stat{{Name: "test", Labels: map[string]string{"response_code": "404"}, Value: 200}},
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name: "test",
						Dimensions: map[string]string{
							"response_code": `response_code`,
						},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "test",
						Value: `200`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{}),
			},
			wantErr: true,
		},
		{
			args: args{
				conf: []*MetricConfig{
					{
						Name:       "test",
						Dimensions: map[string]string{},
					},
				},
				definition: []*MetricDefinition{
					{
						Name:  "test",
						Value: `response_code`,
					},
				},
				bag: attribute.NewMutableBagForMap(map[string]interface{}{}),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := newMetrics(tt.args.conf, tt.args.definition)
			if (err != nil) != tt.wantErr {
				t.Errorf("newMetric() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if metric == nil {
				return
			}
			got, err := metric.Stat(tt.args.bag)
			if (err != nil) != tt.wantStatErr {
				t.Errorf("Stat() error = %v, wantStatErr %v", err, tt.wantStatErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Stat() got = %v, want %v", got, tt.want)
			}
		})
	}
}
