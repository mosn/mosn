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

package ext

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/parser"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

const (
	pick = "pick"
)

var (
	Macros = cel.Macros(
		parser.NewGlobalMacro("conditional", 3, conditionalExpander),
		parser.NewGlobalVarArgMacro(pick, pickExpander))
)

func conditionalExpander(eh parser.ExprHelper, target *expr.Expr, args []*expr.Expr) (*expr.Expr, *common.Error) {
	if target != nil {
		return nil, &common.Error{
			Message:  "unexpected target in conditional",
			Location: eh.OffsetLocation(target.Id)}
	}

	// Convert conditional to a ternary operator.
	return eh.GlobalCall(operators.Conditional, args...), nil
}

func pickExpander(eh parser.ExprHelper, target *expr.Expr, args []*expr.Expr) (*expr.Expr, *common.Error) {
	if target != nil {
		return nil, &common.Error{
			Message:  "unexpected target in pick()",
			Location: eh.OffsetLocation(target.Id)}
	}

	out := args[len(args)-1]
	var selector *expr.Expr
	for i := len(args) - 2; i >= 0; i-- {
		selector = nil
		switch lhs := args[i].ExprKind.(type) {
		case *expr.Expr_SelectExpr:
			if lhs.SelectExpr.TestOnly {
				return nil, &common.Error{
					Message:  "unsupported use of has() macro in pick()",
					Location: eh.OffsetLocation(args[i].Id)}
			}
			// a.f | x --> has(a.f) ? a.f : x
			selector = eh.PresenceTest(lhs.SelectExpr.Operand, lhs.SelectExpr.Field)
		case *expr.Expr_CallExpr:
			if lhs.CallExpr.Function == operators.Index {
				// a["f"] | x --> "f" in a ? a["f"] : x
				selector = eh.GlobalCall(operators.In, lhs.CallExpr.Args[1], lhs.CallExpr.Args[0])
			}
		}

		// otherwise, a | b --> a
		if selector == nil {
			out = args[i]
		} else {
			out = eh.GlobalCall(operators.Conditional, selector, args[i], out)
		}
	}

	return out, nil
}
