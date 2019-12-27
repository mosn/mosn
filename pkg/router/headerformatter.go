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
	"strings"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

func getHeaderFormatter(value string, append bool) headerFormatter {
	// TODO: variable headers would be support very soon
	if strings.Index(value, "%") != -1 {
		log.DefaultLogger.Warnf("variable headers not support yet, skip, value: %s", value)
		return nil
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

func (f *plainHeaderFormatter) format(requestInfo types.RequestInfo) string {
	return f.staticValue
}
