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

package cel

import (
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/common/debug"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/cel/ext"
)

type expression struct {
	expr     *expr.Expr
	provider *attributeProvider
	program  cel.Program
}

func (ex *expression) Evaluate(bag attribute.Bag) (out interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during evaluation: %v", r)
		}
	}()

	result, _, err := ex.program.Eval(ex.provider.newActivation(bag))
	if err != nil {
		return nil, err
	}
	out, err = ext.RecoverValue(result)
	return
}

func (ex *expression) String() string {
	return debug.ToDebugString(ex.expr)
}
