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

package rules

import "mosn.io/mosn/pkg/log"

func init() {
	MustRegister(SimpleMatcher, &simpleMatcher{})
}

// macther factory
var mactherFactory = make(map[string]TransferMatcher)

func MustRegister(typ string, matcher TransferMatcher) {
	if mactherFactory[typ] != nil {
		log.DefaultLogger.Fatalf("[stream filter][transcoder][rules]target stream macther already exists: %s", typ)
	}
	mactherFactory[typ] = matcher
}

func GetMatcher(typ string) TransferMatcher {
	return mactherFactory[typ]
}
