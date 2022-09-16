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

package holmes

import (
	"testing"

	"mosn.io/holmes"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
)

func TestNotInit(t *testing.T) {
	var options []holmes.Option
	err := SetOptions(options)
	if err == nil {
		t.Fatal("set options before init not failed")
	}
}

func TestSetOptions(t *testing.T) {
	// set for unit test
	origPath := types.MosnBasePath
	types.MosnBasePath = "/tmp"
	defer func() {
		types.MosnBasePath = origPath
	}()

	var options []holmes.Option
	err := SetOptions(options)
	if err == nil {
		t.Fatal("set options before init not failed")
	}

	Register(&v2.MOSNConfig{})

	buf := []byte(cfg)
	if err := OnHolmesPluginParsed(buf); err != nil {
		t.Fatalf("OnHolmesPluginParsed failed: %v", err)
	}

	if err := SetOptions(options); err != nil {
		t.Fatalf("set options failed: %v", err)
	}
}

const cfg = `
{
	"enable": true,
	"binarydump": true,
	"fullstackdump": true,
	"collectinterval": "10s",
	"cpumax": 50,
	"cpuprofile": {
		"enable": true,
		"triggermin": 10,
		"triggerabs": 90,
		"triggerdiff": 10,
		"cooldown": "1m"
	},
	"memoryprofile": {
		"enable": true,
		"triggermin": 10,
		"triggerabs": 90,
		"triggerdiff": 10,
		"cooldown": "1m"
	},
	"gcheapprofile": {
		"enable": true,
		"triggermin": 10,
		"triggerabs": 90,
		"triggerdiff": 10,
		"cooldown": "1m"
	},
	"goroutineprofile": {
		"enable": true,
		"triggermin": 1000,
		"triggerabs": 9000,
		"triggerdiff": 10,
		"goroutinetriggernummax": 10000,
		"cooldown": "1m"
	},
	"threadprofile": {
		"enable": true,
		"triggermin": 1000,
		"triggerabs": 9000,
		"triggerdiff": 10,
		"cooldown": "1m"
	},
	"shrinkthread": {
		"enable": true,
		"threshold": 100,
		"delay": "10m"
	}
}
`
