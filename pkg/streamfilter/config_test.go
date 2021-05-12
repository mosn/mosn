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

package streamfilter

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	monkey "github.com/cch123/supermonkey"
	"github.com/golang/mock/gomock"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
)

func TestLoadAndRegisterStreamFiltersJson(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer monkey.UnpatchAll()

	tmpFile, err := ioutil.TempFile(os.TempDir(), "*.json")
	if err != nil {
		t.Errorf("Cannot create temporary file, err: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte(`
{
  "servers": [],
  "cluster_manager": {},
  "admin": {},
  "stream_filters": [
    {
      "name": "filterConfig1",
      "stream_filters": [
        {
          "type": "demo",
          "config": {
            "User": "user1"
          }
        }
      ]
    },
    {
      "name": "filterConfig2",
      "stream_filters": [
        {
          "type": "demo",
          "config": {
            "User": "user2"
          }
        }
      ]
    }
  ]
}
`)
	if _, err = tmpFile.Write(text); err != nil {
		t.Fatalf("Failed to write to temporary file, err: %v", err)
	}

	addConfigCall := false
	monkey.Patch(GetStreamFilterManager, func() StreamFilterManager {
		filterManager := NewMockStreamFilterManager(ctrl)
		filterManager.EXPECT().AddOrUpdateStreamFilterConfig(gomock.Any(), gomock.Any()).Do(func(key string, config StreamFiltersConfig) error {
			addConfigCall = true
			return nil
		}).AnyTimes()
		return filterManager
	})

	LoadAndRegisterStreamFilters(tmpFile.Name())

	if !addConfigCall {
		t.Errorf("LoadAndRegisterStreamFilters failed")
	}
}

func TestLoadAndRegisterStreamFiltersYaml(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer monkey.UnpatchAll()

	tmpFile, err := ioutil.TempFile(os.TempDir(), "*.yaml")
	if err != nil {
		t.Errorf("Cannot create temporary file, err: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	text := []byte(`
---
servers: []
cluster_manager: {}
admin: {}
stream_filters:
- name: filterConfig1
  stream_filters:
  - type: demo
    config:
      User: user1
- name: filterConfig2
  stream_filters:
  - type: demo
    config:
      User: user2
`)
	if _, err = tmpFile.Write(text); err != nil {
		t.Fatalf("Failed to write to temporary file, err: %v", err)
	}

	addConfigCall := false
	monkey.Patch(GetStreamFilterManager, func() StreamFilterManager {
		filterManager := NewMockStreamFilterManager(ctrl)
		filterManager.EXPECT().AddOrUpdateStreamFilterConfig(gomock.Any(), gomock.Any()).Do(func(key string, config StreamFiltersConfig) error {
			addConfigCall = true
			return nil
		}).AnyTimes()
		return filterManager
	})

	LoadAndRegisterStreamFilters(tmpFile.Name())

	if !addConfigCall {
		t.Errorf("LoadAndRegisterStreamFilters failed")
	}
}

func TestRegisterStreamFilters(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	defer monkey.UnpatchAll()

	config := []StreamFilters{
		{
			// empty name, should failed
			Name:    "",
			Filters: []v2.Filter{{Type: "type1"}, {Type: "type2"}},
		},
		{
			// should success
			Name:    "name",
			Filters: []v2.Filter{{Type: "type1"}, {Type: "type2"}},
		},
		{
			// empty filters, should failed
			Name:    "name",
			Filters: []v2.Filter{},
		},
	}

	addConfigCount := 0
	monkey.Patch(GetStreamFilterManager, func() StreamFilterManager {
		filterManager := NewMockStreamFilterManager(ctrl)
		filterManager.EXPECT().AddOrUpdateStreamFilterConfig(gomock.Any(), gomock.Any()).Do(func(key string, config StreamFiltersConfig) error {
			addConfigCount++
			return nil
		}).AnyTimes()
		return filterManager
	})

	RegisterStreamFilters(config)

	if addConfigCount != 1 {
		t.Errorf("RegisterStreamFilters fail, count=%v want=1", addConfigCount)
	}
}

func TestCreateStreamFilterFactoryFromConfig(t *testing.T) {
	api.RegisterStream("test1", func(cfg map[string]interface{}) (api.StreamFilterChainFactory, error) {
		return &struct {
			api.StreamFilterChainFactory
		}{}, nil
	})
	api.RegisterStream("test_nil", func(cfg map[string]interface{}) (api.StreamFilterChainFactory, error) {
		return nil, nil
	})
	api.RegisterStream("test_error", func(cfg map[string]interface{}) (api.StreamFilterChainFactory, error) {
		return nil, errors.New("invalid factory create")
	})
	sff := createStreamFilterFactoryFromConfig([]v2.Filter{
		{Type: "test1"},
		{Type: "test_error"},
		{Type: "not registered"},
		{Type: "test_nil"},
	})
	if len(sff) != 1 {
		t.Fatalf("expected got only one success factory, but got %d", len(sff))
	}
}
