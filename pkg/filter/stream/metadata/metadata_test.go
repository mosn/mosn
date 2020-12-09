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

package metadata

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/protocol"
	_ "mosn.io/mosn/pkg/proxy"
	"mosn.io/pkg/buffer"
)

func init() {
	RegisterDriver("IDC", &MockIdcDriver{})
	RegisterDriver("ZONE", &MockZoneDriver{})
}

func TestCreateMetadataFilterFactory(t *testing.T) {
	m := map[string]interface{}{}
	if f, err := CreateMetadataFilterFactory(m); f == nil || err != nil {
		t.Errorf("CreateMetadataFilterFactory failed: %v.", err)
	}
}

func TestMetadataNewStreamFilter(t *testing.T) {

	idcKey := "IDC"
	idcValue := "test-idc"
	zoneKey := "ZONE"
	zoneValue := "test-zone"

	data, err := mockConfig()
	if err != nil {
		t.Fatalf("mock metadata config failed: %v", err)
	}

	// set env
	os.Setenv(idcKey, idcValue)
	os.Setenv(zoneKey, zoneValue)

	fa, err := CreateMetadataFilterFactory(data)
	if err != nil {
		t.Fatalf("CreateMetadataFilterFactory failed: %v", err)
	}

	f := NewMetadataFilter(fa.(*FilterConfigFactory))
	reqHeaders := protocol.CommonHeader(map[string]string{})

	f.OnReceive(context.TODO(), reqHeaders, nil, nil)

	if v, ok := reqHeaders.Get(metadataPrefix + idcKey); !ok || v != idcValue {
		t.Errorf("get idc info failed: got %v want %v", v, idcValue)
	}

	if v, ok := reqHeaders.Get(metadataPrefix + zoneKey); !ok || v != zoneValue {
		t.Errorf("get zone info failed: got %v want %v", v, zoneValue)
	}
}

func BenchmarkMetadata(b *testing.B) {
	idcKey := "IDC"
	idcValue := "test-idc"
	zoneKey := "ZONE"
	zoneValue := "test-zone"

	data, err := mockConfig()
	if err != nil {
		b.Fatalf("mock metadata config failed: %v", err)
	}

	// set env
	os.Setenv(idcKey, idcValue)
	os.Setenv(zoneKey, zoneValue)

	fa, err := CreateMetadataFilterFactory(data)
	if err != nil {
		b.Fatalf("CreateMetadataFilterFactory failed: %v", err)
	}

	f := NewMetadataFilter(fa.(*FilterConfigFactory))

	for i := 0; i < b.N; i++ {
		reqHeaders := protocol.CommonHeader(map[string]string{})
		f.OnReceive(context.TODO(), reqHeaders, nil, nil)

		if v, ok := reqHeaders.Get(metadataPrefix + idcKey); !ok || v != idcValue {
			b.Errorf("get idc info failed: got %v want %v", v, idcValue)
		}

		if v, ok := reqHeaders.Get(metadataPrefix + zoneKey); !ok || v != zoneValue {
			b.Errorf("get zone info failed: got %v want %v", v, zoneValue)
		}
	}
}

func mockConfig() (map[string]interface{}, error) {
	mockConfig := `{"disable": false,
     "case_sensitive": true,
     "metadataers":[
         {
             "meta_data_key":"IDC",
              "config":{
                  "mock_env_name_list":["IDC"]
              }
         },
         {
             "meta_data_key":"ZONE",
              "config":{
                  "mock_env_name_list":["ZONE"]
              }
         }
      ]}`

	data := map[string]interface{}{}
	err := json.Unmarshal([]byte(mockConfig), &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

type MockConfig struct {
	MockEnvNameList []string `json:"mock_env_name_list,omitempty"`
}

type MockIdcDriver struct {
	MockDriver
}

type MockZoneDriver struct {
	MockDriver
}

type MockDriver struct {
	Name string
}

func (d *MockDriver) Init(cfg map[string]interface{}) error {
	ic := &MockConfig{}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, ic); err != nil {
		return err
	}

	for _, name := range ic.MockEnvNameList {
		d.Name = os.Getenv(name)
		if d.Name != "" {
			return nil
		}
	}
	if d.Name == "" {
		return errors.New("get idc name is nil, idc env name: " + strings.Join(ic.MockEnvNameList, ","))

	}
	return nil
}

func (d *MockDriver) BuildMetadata(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) (string, error) {
	return d.Name, nil
}
