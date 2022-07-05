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

package matcher

import (
	"mosn.io/mosn/pkg/log"
)

type MatcherFactory func(config interface{}) RuleMatcher

// matcher factory
var matcherFactoryMaps = make(map[string]MatcherFactory)

func RegisterMatcherFatcory(typ string, factory MatcherFactory) {
	if matcherFactoryMaps[typ] != nil {
		log.DefaultLogger.Fatalf("[stream filter][transcoder][matcher]target stream matcher already exists: %s", typ)
	}
	matcherFactoryMaps[typ] = factory
}

func NewMatcher(cfg *MatcherConfig) RuleMatcher {
	mf := matcherFactoryMaps[cfg.MatcherType]
	if mf == nil {
		log.DefaultLogger.Errorf("[stream filter][transcoder][matcher]target stream matcher not exists: %s", cfg.MatcherType)
		return nil
	}
	return mf(cfg.Config)
}
