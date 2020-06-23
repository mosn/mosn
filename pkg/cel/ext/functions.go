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
	"regexp"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/interpreter/functions"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"mosn.io/mosn/pkg/cel/attribute"
)

var (
	StandardFunctions = cel.Declarations([]*expr.Decl{
		decls.NewFunction("match",
			decls.NewOverload("match",
				[]*expr.Type{decls.String, decls.String}, decls.Bool),
			decls.NewInstanceOverload("match",
				[]*expr.Type{decls.String, decls.String}, decls.Bool)),
		decls.NewFunction("reverse",
			decls.NewOverload("reverse",
				[]*expr.Type{decls.String}, decls.String),
			decls.NewInstanceOverload("reverse",
				[]*expr.Type{decls.String}, decls.String)),
		decls.NewFunction("toLower",
			decls.NewOverload("toLower",
				[]*expr.Type{decls.String}, decls.String),
			decls.NewInstanceOverload("toLower",
				[]*expr.Type{decls.String}, decls.String)),
		decls.NewFunction("toUpper",
			decls.NewOverload("toUpper",
				[]*expr.Type{decls.String}, decls.String),
			decls.NewInstanceOverload("toUpper",
				[]*expr.Type{decls.String}, decls.String)),
		decls.NewFunction("conditionalString",
			decls.NewOverload("conditionalString",
				[]*expr.Type{decls.Bool, decls.String, decls.String}, decls.String)),
		decls.NewFunction("email",
			decls.NewOverload("email",
				[]*expr.Type{decls.String}, decls.NewObjectType(emailAddressType))),
		decls.NewFunction("dnsName",
			decls.NewOverload("dnsName",
				[]*expr.Type{decls.String}, decls.NewObjectType(dnsType))),
		decls.NewFunction("uri",
			decls.NewOverload("uri",
				[]*expr.Type{decls.String}, decls.NewObjectType(uriType))),
		decls.NewFunction("ip",
			decls.NewOverload("ip",
				[]*expr.Type{decls.String}, decls.NewObjectType(ipAddressType))),
		decls.NewFunction("emptyStringMap",
			decls.NewOverload("emptyStringMap",
				[]*expr.Type{}, stringMapType)),
	}...)

	StandardOverloads = cel.Functions([]*functions.Overload{
		{Operator: "match",
			Binary: callInStrStrOutBool(externMatch),
		},
		{Operator: "matches_string",
			OperandTrait: traits.MatcherType,
			Binary:       callInStrStrOutBoolWithErr(regexp.MatchString),
		},
		{Operator: "reverse",
			Unary: callInStrOutStr(externReverse),
		},
		{Operator: "toLower",
			Unary: callInStrOutStr(strings.ToLower),
		},
		{Operator: "toUpper",
			Unary: callInStrOutStr(strings.ToUpper),
		},
		{Operator: "conditionalString",
			Function: callInBoolStrStrOutString(externConditionalString),
		},
		{Operator: "email",
			Unary: func(v ref.Val) ref.Val {
				if v.Type() != types.StringType {
					return types.NewErr("overload cannot be applied to '%s'", v.Type())
				}
				out, err := externEmail(v.Value().(string))
				if err != nil {
					return types.NewErr(err.Error())
				}
				return wrapperValue{typ: attribute.EMAIL_ADDRESS, s: out}
			}},
		{Operator: "dnsName",
			Unary: func(v ref.Val) ref.Val {
				if v.Type() != types.StringType {
					return types.NewErr("overload cannot be applied to '%s'", v.Type())
				}
				out, err := externDNSName(v.Value().(string))
				if err != nil {
					return types.NewErr(err.Error())
				}
				return wrapperValue{typ: attribute.DNS_NAME, s: out}
			}},
		{Operator: "uri",
			Unary: func(v ref.Val) ref.Val {
				if v.Type() != types.StringType {
					return types.NewErr("overload cannot be applied to '%s'", v.Type())
				}
				out, err := externURI(v.Value().(string))
				if err != nil {
					return types.NewErr(err.Error())
				}
				return wrapperValue{typ: attribute.URI, s: out}
			}},
		{Operator: "ip",
			Unary: func(v ref.Val) ref.Val {
				if v.Type() != types.StringType {
					return types.NewErr("overload cannot be applied to '%s'", v.Type())
				}
				out, err := externIP(v.Value().(string))
				if err != nil {
					return types.NewErr(err.Error())
				}
				return wrapperValue{typ: attribute.IP_ADDRESS, bytes: out}
			}},
		{Operator: "emptyStringMap",
			Function: func(args ...ref.Val) ref.Val {
				if len(args) != 0 {
					return types.NewErr("emptyStringMap takes no arguments")
				}
				return emptyStringMap
			}},
	}...)
)

func callInStrOutStr(fn func(string) string) functions.UnaryOp {
	return func(val ref.Val) ref.Val {
		vVal, ok := val.(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(val)
		}
		out := fn(string(vVal))
		return types.String(out)
	}
}

func callInStrStrOutBool(fn func(string, string) bool) functions.BinaryOp {
	return func(val, arg ref.Val) ref.Val {
		vVal, ok := val.(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(val)
		}
		argVal, ok := arg.(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(arg)
		}
		out := fn(string(vVal), string(argVal))
		return types.Bool(out)
	}
}

func callInStrStrOutBoolWithErr(fn func(string, string) (bool, error)) functions.BinaryOp {
	return func(val, arg ref.Val) ref.Val {
		vVal, ok := val.(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(val)
		}
		argVal, ok := arg.(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(arg)
		}
		out, err := fn(string(vVal), string(argVal))
		if err != nil {
			return types.NewErr(err.Error())
		}
		return types.Bool(out)
	}
}

func callInBoolStrStrOutString(fn func(bool, string, string) string) functions.FunctionOp {
	return func(args ...ref.Val) ref.Val {
		if len(args) != 3 {
			return types.NoSuchOverloadErr()
		}
		vVal, ok := args[0].(types.Bool)
		if !ok {
			return types.MaybeNoSuchOverloadErr(args[0])
		}
		arg1Val, ok := args[1].(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(args[1])
		}
		arg2Val, ok := args[2].(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(args[2])
		}
		out := fn(bool(vVal), string(arg1Val), string(arg2Val))
		return types.String(out)
	}
}
