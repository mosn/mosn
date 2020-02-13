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

package filter

import (
	"context"
	"errors"
	"testing"

	"mosn.io/api"
)

type testStreamFilterFactory struct{}

func (f *testStreamFilterFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
}
func testStreamFilterFactoryCreator(config map[string]interface{}) (api.StreamFilterChainFactory, error) {
	if _, ok := config["error"]; ok {
		return nil, errors.New("error")
	}
	return &testStreamFilterFactory{}, nil
}

type testNetworkFilterFactory struct{}

func (f *testNetworkFilterFactory) CreateFilterChain(context context.Context, callbacks api.NetWorkFilterChainFactoryCallbacks) {
}
func testNetworkFilterFactoryCreator(config map[string]interface{}) (api.NetworkFilterChainFactory, error) {
	if _, ok := config["error"]; ok {
		return nil, errors.New("error")
	}
	return &testNetworkFilterFactory{}, nil
}

func TestCreateStreamFilterChainFactory(t *testing.T) {
	name := "test"
	api.RegisterStream(name, testStreamFilterFactoryCreator)
	config := make(map[string]interface{})
	if _, err := api.CreateStreamFilterChainFactory("no", config); err == nil {
		t.Error("no register type should return an error")
	}
	if _, err := api.CreateStreamFilterChainFactory(name, config); err != nil {
		t.Error(err)
	}
	config["error"] = true
	if _, err := api.CreateStreamFilterChainFactory(name, config); err == nil {
		t.Error("create factory failed, expected an error")
	}
}
func TestCreateNetworkFilterChainFactory(t *testing.T) {
	name := "test"
	api.RegisterNetwork(name, testNetworkFilterFactoryCreator)
	config := make(map[string]interface{})
	if _, err := api.CreateNetworkFilterChainFactory("no", config); err == nil {
		t.Error("no register type should return an error")
	}
	if _, err := api.CreateNetworkFilterChainFactory(name, config); err != nil {
		t.Error(err)
	}
	config["error"] = true
	if _, err := api.CreateNetworkFilterChainFactory(name, config); err == nil {
		t.Error("create factory failed, expected an error")
	}
}
