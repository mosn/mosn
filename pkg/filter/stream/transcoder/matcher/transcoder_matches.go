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
	"context"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

func init() {
	MustRegister(DefaultMatches)
}

type TransCoderMatchesFunc func(ctx context.Context, header api.HeaderMap, rules []*TransferRule) (*RuleInfo, bool)

var TransCoderMatches TransCoderMatchesFunc

func MustRegister(matches TransCoderMatchesFunc) {
	TransCoderMatches = matches
}

func DefaultMatches(ctx context.Context, header api.HeaderMap, rules []*TransferRule) (*RuleInfo, bool) {
	for _, rule := range rules {
		if rule.Macther == nil {
			continue
		}
		if rule.Macther.Matches(ctx, header) {
			return rule.RuleInfo, true
		}
	}
	if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
		log.DefaultLogger.Debugf("[stream filter][transcoder] no match, matcher %+v", rules)
	}
	return nil, false
}
