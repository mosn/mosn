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
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/cel"
	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/cel/extract"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

var dslRouterCompiler = cel.NewExpressionBuilder(extract.Attributemanifest, cel.CompatCEXL)

type DslExpressionRouteRuleImpl struct {
	*RouteRuleImplBase
	DslExpressions     []attribute.Expression
	originalExpression []v2.DslExpressionMatcher
}

func (drri *DslExpressionRouteRuleImpl) HeaderMatchCriteria() api.KeyValueMatchCriteria {
	return nil
}

func (drri *DslExpressionRouteRuleImpl) PathMatchCriterion() api.PathMatchCriterion {
	return drri
}

func (drri *DslExpressionRouteRuleImpl) RouteRule() api.RouteRule {
	return drri
}

func (drri *DslExpressionRouteRuleImpl) Matcher() string {
	return ""
}

func (drri *DslExpressionRouteRuleImpl) MatchType() api.PathMatchType {
	return api.Variable
}

func (drri *DslExpressionRouteRuleImpl) FinalizeRequestHeaders(ctx context.Context, headers api.HeaderMap, requestInfo api.RequestInfo) {
}

func (drri *DslExpressionRouteRuleImpl) Match(ctx context.Context, headers api.HeaderMap) api.Route {
	parentBag := extract.ExtractAttributes(ctx, headers, nil, nil, nil, nil, time.Now())
	bag := attribute.NewMutableBag(parentBag)
	bag.Set(extract.KContext, ctx)
	for i, dslExpression := range drri.DslExpressions {
		res, err := dslExpression.Evaluate(bag)
		if err != nil {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf(RouterLogFormat, "dsl route rule", "match failed", err, drri.originalExpression[i])
			}
			return nil
		}
		if !res.(bool) {
			if log.DefaultLogger.GetLogLevel() >= log.DEBUG {
				log.DefaultLogger.Debugf(RouterLogFormat, "dsl route rule", "not match", drri.originalExpression[i])
			}
			return nil
		}
	}
	return drri
}

func parseConfigToDslExpression(dslExpressions []v2.DslExpressionMatcher) []attribute.Expression {
	expressions := make([]attribute.Expression, 0, len(dslExpressions))
	for i := range dslExpressions {
		if dslExpressions[i].Expression != "" {
			expr, _, err := dslRouterCompiler.Compile(dslExpressions[i].Expression)
			if err != nil {
				log.DefaultLogger.Errorf(RouterLogFormat, "dsl router rule", "parseConfigToDslExpression", "expression compile failed", dslExpressions[i].Expression)
				continue
			}
			expressions = append(expressions, expr)
		}
	}
	return expressions
}
