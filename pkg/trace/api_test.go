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

package trace

import (
	"github.com/stretchr/testify/assert"
	"mosn.io/api"

	"testing"

	"mosn.io/mosn/pkg/types"
)

func TestAPI(t *testing.T) {
	driver := NewDefaultDriverImpl()
	proto := types.ProtocolName("test")

	driver.Register(proto, func(config map[string]interface{}) (api.Tracer, error) {
		return &mockTracer{}, nil
	})

	driver.Init(nil)
	RegisterDriver("driverdriver", driver)

	err := Init("driverdriver", map[string]interface{}{})
	assert.Nil(t, err)
}

func TestEnable(t *testing.T) {
	Enable()
	assert.True(t, IsEnabled())

	Disable()
	assert.False(t, IsEnabled())
}
