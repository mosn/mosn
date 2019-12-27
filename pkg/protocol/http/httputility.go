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

package http

import (
	"strings"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// the query string looks like:  "field1=value1&field2=value2&field3=value3..."
func ParseQueryString(query string) types.QueryParams {
	var QueryParams = make(types.QueryParams, 10)

	if "" == query {
		return QueryParams
	}
	queryMaps := strings.Split(query, "&")

	for _, qm := range queryMaps {
		queryMap := strings.Split(qm, "=")

		if len(queryMap) != 2 {
			log.DefaultLogger.Errorf("parse query parameters error,parameters = %s", qm)
		} else {
			QueryParams[strings.TrimSpace(queryMap[0])] = strings.TrimSpace(queryMap[1])
		}
	}

	return QueryParams
}
