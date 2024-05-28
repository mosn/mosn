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

package router

import (
	"context"
	"strings"

	"mosn.io/pkg/variable"
)

func getHeaderFormatter(value string, append bool) headerFormatter {
	if len(value) > 2 && strings.HasPrefix(value, "%") && strings.HasSuffix(value, "%") {
		variableName := strings.Trim(value, "%")
		// todo cache the variable so we don't need to find it in format method
		if _, err := variable.Check(variableName); err == nil {
			return &variableHeaderFormatter{
				isAppend:     append,
				variableName: variableName,
			}
		}
	}
	return &plainHeaderFormatter{
		isAppend:    append,
		staticValue: value,
	}
}

type plainHeaderFormatter struct {
	isAppend    bool
	staticValue string
}

func (f *plainHeaderFormatter) append() bool {
	return f.isAppend
}

func (f *plainHeaderFormatter) format(_ context.Context) string {
	return f.staticValue
}

type variableHeaderFormatter struct {
	isAppend     bool
	variableName string
}

func (v *variableHeaderFormatter) format(ctx context.Context) string {
	value, err := variable.GetString(ctx, v.variableName)
	if err != nil {
		return ""
	}
	return value
}

func (v *variableHeaderFormatter) append() bool {
	return v.isAppend
}
