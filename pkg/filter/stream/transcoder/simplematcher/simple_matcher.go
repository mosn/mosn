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

package simplematcher

import (
	"context"
	"mosn.io/mosn/pkg/filter/stream/transcoder/matcher"
	"mosn.io/mosn/pkg/types"
)

const SimpleMatcherFactoryKey = "simpleMatcher"

func init() {
	matcher.RegisterMatcherFatcory(SimpleMatcherFactoryKey, SimpleMatcherFactory)
}

type SimpleRuleMatcher struct {
	config interface{}
}

func (hrm *SimpleRuleMatcher) Matches(ctx context.Context, headers types.HeaderMap) bool {
	return true
}

func SimpleMatcherFactory(config interface{}) matcher.RuleMatcher {
	return &SimpleRuleMatcher{
		config: config,
	}
}
