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
	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/cel/cexl"
	"mosn.io/mosn/pkg/cel/ext"
)

// LanguageMode controls parsing and evaluation properties of the expression builder
type LanguageMode int

const (
	// CEL mode uses CEL syntax and runtime
	CEL LanguageMode = iota

	// CompatCEXL uses CEXL syntax and CEL runtime
	CompatCEXL
)

// ExpressionBuilder creates a CEL interpreter from an attribute manifest.
type ExpressionBuilder struct {
	mode     LanguageMode
	provider *attributeProvider
	env      *cel.Env
	opt      cel.ProgramOption
}

// NewExpressionBuilder returns a new ExpressionBuilder
func NewExpressionBuilder(attributes map[string]attribute.Kind, mode LanguageMode) *ExpressionBuilder {
	provider := newAttributeProvider(attributes)
	env := provider.newEnvironment()

	return &ExpressionBuilder{
		mode:     mode,
		provider: provider,
		env:      env,
		opt:      ext.StandardOverloadsEnvOption(),
	}
}

// Compile the given text and return a pre-compiled expression object.
func (e *ExpressionBuilder) check(text string) (checked *cel.Ast, typ attribute.Kind, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during CEL parsing of expression %q", text)
		}
	}()

	typ = attribute.VALUE_TYPE_UNSPECIFIED

	if e.mode == CompatCEXL {
		text, err = cexl.SourceCEXLToCEL(text)
		if err != nil {
			return
		}
	}

	parsed, iss := e.env.Parse(text)
	if iss != nil && iss.Err() != nil {
		err = iss.Err()
		return
	}

	checked, iss = e.env.Check(parsed)
	if iss != nil && iss.Err() != nil {
		err = iss.Err()
		return
	}

	typ = ext.RecoverType(checked.ResultType())
	return
}

// Compile the given text and return a pre-compiled expression object.
func (e *ExpressionBuilder) Compile(text string) (attribute.Expression, attribute.Kind, error) {
	checked, typ, err := e.check(text)
	if err != nil {
		return nil, typ, err
	}

	program, err := e.env.Program(checked, e.opt)
	if err != nil {
		return nil, typ, err
	}

	return &expression{
		provider: e.provider,
		expr:     checked.Expr(),
		program:  program,
	}, typ, nil
}

// EvalType returns the type of an expression
func (e *ExpressionBuilder) EvalType(text string) (attribute.Kind, error) {
	_, typ, err := e.check(text)
	return typ, err
}
