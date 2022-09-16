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
	"context"
	"errors"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

var ErrNoSuchDriver = errors.New("no such driver")

type globalHolder struct {
	enable bool
	driver api.Driver
}

var global = globalHolder{
	enable: false,
}

func SpanFromContext(ctx context.Context) api.Span {
	if val, err := variable.Get(ctx, types.VariableTraceSpan); err == nil {
		if sp, ok := val.(api.Span); ok {
			return sp
		}
	}
	return nil
}

func Init(typ string, config map[string]interface{}) error {
	if driver, ok := drivers[typ]; ok {
		err := driver.Init(config)
		if err != nil {
			return err
		}
		global.driver = driver
		global.enable = true
		return nil
	} else {
		return ErrNoSuchDriver
	}
}

func Enable() {
	if global.driver == nil {
		return
	}
	global.enable = true
}

func Disable() {
	global.enable = false
}

func IsEnabled() bool {
	return global.enable
}

func Tracer(protocol types.ProtocolName) api.Tracer {
	return global.driver.Get(protocol)
}

func Driver() api.Driver {
	return global.driver
}
