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

package flowcontrol

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/stretchr/testify/assert"
)

func TestStreamFilterFactory(t *testing.T) {
	mockConfig := `{
    "global_switch": false,
    "monitor": false,
    "limit_key_type": "PATH",
    "action": {
        "status": 0,
        "body": ""
    },
    "rules": [
        {
            "resource": "/http",
            "limitApp": "",
            "grade": 1,
            "threshold": 1,
            "strategy": 0
        }
    ]
}`
	data := map[string]interface{}{}
	err := json.Unmarshal([]byte(mockConfig), &data)
	assert.Nil(t, err)
	f, err := createRpcFlowControlFilterFactory(data)
	assert.NotNil(t, f)
	assert.Nil(t, err)

	// invalidCfg
	mockConfig = `{
    "global_switch": false,
    "monitor": false,
    "limit_key_type": "unknown",
    "action": {
        "status": 0,
        "body": ""
    },
    "rules": [
        {
            "resource": "/http",
            "limitApp": "",
            "grade": 1,
            "threshold": 1,
            "strategy": 0
        }
    ]
}`
	err = json.Unmarshal([]byte(mockConfig), &data)
	assert.Nil(t, err)
	f, err = createRpcFlowControlFilterFactory(data)
	assert.Nil(t, f)
	assert.NotNil(t, err)
}

func TestIsValidCfg(t *testing.T) {
	validConfig := `{
    "global_switch": false,
    "monitor": false,
    "limit_key_type": "PATH",
    "action": {
        "status": 0,
        "body": ""
    },
    "rules": [
        {
            "resource": "/http",
            "limitApp": "",
            "grade": 1,
            "threshold": 1,
            "strategy": 0
        }
    ]
}`
	cfg := &Config{}
	err := json.Unmarshal([]byte(validConfig), cfg)
	assert.Nil(t, err)
	isValid, err := isValidConfig(cfg)
	assert.True(t, isValid)
	assert.Nil(t, err)

	// invalid
	invalidCfg := `{
    "global_switch": false,
    "monitor": false,
    "limit_key_type": "????",
    "action": {
        "status": 0,
        "body": ""
    },
    "rules": [
        {
            "resource": "/http",
            "limitApp": "",
            "grade": 1,
            "threshold": 1,
            "strategy": 0
        }
    ]
}`
	err = json.Unmarshal([]byte(invalidCfg), cfg)
	assert.Nil(t, err)
	isValid, err = isValidConfig(cfg)
	assert.False(t, isValid)
	assert.NotNil(t, err)
}

func Test_parseTrafficType(t *testing.T) {
	type args struct {
		conf map[string]interface{}
	}
	tests := []struct {
		name string
		args args
		want base.TrafficType
	}{
		{
			name: "default-inbound",
			args: struct{ conf map[string]interface{} }{},
			want: base.Inbound,
		}, {
			name: "outbound",
			args: struct{ conf map[string]interface{} }{
				map[string]interface{}{"direction": "outbound"}},
			want: base.Outbound,
		}, {
			name: "inbound",
			args: struct{ conf map[string]interface{} }{
				map[string]interface{}{"direction": "inbound"}},
			want: base.Inbound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseTrafficType(tt.args.conf); got != tt.want {
				t.Errorf("parseTrafficType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_loadConfig(t *testing.T) {
	type args struct {
		conf map[string]interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    *Config
		wantErr bool
	}{
		{
			name:    "nil config return default config",
			args:    struct{ conf map[string]interface{} }{conf: nil},
			want:    defaultConfig(),
			wantErr: false,
		}, {
			name: "nil config return default config",
			args: struct{ conf map[string]interface{} }{conf: map[string]interface{}{
				"app_name": "test",
			}},
			want: func() *Config {
				conf := defaultConfig()
				conf.AppName = "test"
				return conf
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadConfig(tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
