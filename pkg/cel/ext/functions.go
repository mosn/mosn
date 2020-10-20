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
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/interpreter/functions"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"mosn.io/api"
	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/pkg/log"
)

func init() {
	check(RegisterSimpleFunctionBoth("match", externMatch))
	check(RegisterSimpleFunctionBoth("reverse", externReverse))
	check(RegisterSimpleFunctionBoth("toLower", strings.ToLower))
	check(RegisterSimpleFunctionBoth("toUpper", strings.ToUpper))
	check(RegisterSimpleFunctionBoth("conditionalString", externConditionalString))
	check(RegisterSimpleFunctionBoth("email", mail.ParseAddress))
	check(RegisterSimpleFunctionBoth("dnsName", externDNSName))
	check(RegisterSimpleFunctionBoth("uri", url.Parse))
	check(RegisterSimpleFunctionBoth("ip", net.ParseIP))
	check(RegisterSimpleFunctionBoth("string", func(ip net.IP) string {
		return ip.String()
	}))
	check(RegisterSimpleFunctionBoth("string", func(u *url.URL) string {
		return u.String()
	}))

	check(RegisterSimpleFunction("emptyStringMap", externEmptyStringMap))
	// Replace the default string match
	check(RegisterFunction("matches_string", traits.MatcherType, regexp.MatchString))
	// internal operation for MOSN
	// rewrite request url
	check(RegisterSimpleFunctionBoth("rewrite_request_url", rewriteRequestUrl))
	// add request header
	check(RegisterSimpleFunctionBoth("add_request_header", addRequestheader))
	// del request header
	check(RegisterSimpleFunctionBoth("del_request_header", delRequestheader))
	// add reqponse header
	check(RegisterSimpleFunctionBoth("add_response_header", addResponseheader))
	// del response header
	check(RegisterSimpleFunctionBoth("del_response_header", delResponseheader))

}

// function type's input or output parameter count.
const (
	paraZero  int = 0
	paraOne   int = 1
	paraTwo   int = 2
	paraThree int = 3
)

func check(err error) {
	if err == nil {
		return
	}
	log.DefaultLogger.Warnf("%s", err)
}

func StandardFunctionsEnvOption() cel.EnvOption {
	decl := []*expr.Decl{}
	for name, fun := range declsFunc {
		decl = append(decl, decls.NewFunction(name, fun...))
	}
	sort.Slice(decl, func(i, j int) bool {
		return decl[i].Name < decl[j].Name
	})
	return cel.Declarations(decl...)
}

func StandardOverloadsEnvOption() cel.ProgramOption {
	return cel.Functions(functionOverloads...)
}

var declsFunc = map[string][]*expr.Decl_FunctionDecl_Overload{}

func registerOverloadBoth(name string, operator string, fun interface{}, f, i bool) error {
	if !f && !i {
		return nil
	}
	argTypes, resultType, err := getDeclFunc(fun)
	if err != nil {
		return err
	}
	if f {
		declsFunc[name] = append(declsFunc[name], decls.NewOverload(operator, argTypes, resultType))
	}
	if i {
		declsFunc[name] = append(declsFunc[name], decls.NewInstanceOverload(operator, argTypes, resultType))
	}
	return nil
}

func RegisterOverloadBoth(name string, operator string, fun interface{}) error {
	return registerOverloadBoth(name, operator, fun, true, true)
}

func RegisterOverload(name string, operator string, fun interface{}) error {
	return registerOverloadBoth(name, operator, fun, true, false)
}

func RegisterInstanceOverload(name string, operator string, fun interface{}) error {
	return registerOverloadBoth(name, operator, fun, false, true)
}

var functionOverloads []*functions.Overload

func RegisterSimpleFunctionBoth(name string, fun interface{}) error {
	operator := getFunctionTypeUniqueKey(name, fun)
	err := RegisterFunction(operator, 0, fun)
	if err != nil {
		return err
	}
	return RegisterOverloadBoth(name, operator, fun)
}

func RegisterSimpleFunction(name string, fun interface{}) error {
	operator := getFunctionTypeUniqueKey(name, fun)
	err := RegisterFunction(operator, 0, fun)
	if err != nil {
		return err
	}
	return RegisterOverload(name, operator, fun)
}

func RegisterFunction(operator string, trait int, fun interface{}) error {
	wrap, err := wrapFunc(fun)
	if err != nil {
		return err
	}
	o := &functions.Overload{
		Operator:     operator,
		OperandTrait: trait,
	}
	switch fun := wrap.(type) {
	case functions.UnaryOp:
		o.Unary = fun
	case functions.BinaryOp:
		o.Binary = fun
	case functions.FunctionOp:
		o.Function = fun
	}
	functionOverloads = append(functionOverloads, o)
	return nil
}

func getDeclFunc(fun interface{}) (argTypes []*expr.Type, resultType *expr.Type, err error) {
	funVal := reflect.ValueOf(fun)
	if funVal.Kind() != reflect.Func {
		return nil, nil, fmt.Errorf("must be a function, not a %s", funVal.Kind())
	}
	typ := funVal.Type()
	numOut := typ.NumOut()
	switch numOut {
	default:
		return nil, nil, fmt.Errorf("too many result")
	case paraZero:
		return nil, nil, fmt.Errorf("result is required")
	case paraOne, paraTwo:
		resultType = ConvertKind(typ.Out(0))
		if resultType == decls.Null {
			return nil, nil, fmt.Errorf("the result of function %s is unspecified", typ.String())
		}
	}

	numIn := typ.NumIn()
	argTypes = make([]*expr.Type, 0, numIn)
	for i := 0; i != numIn; i++ {
		param := ConvertKind(typ.In(i))
		if param == decls.Null {
			return nil, nil, fmt.Errorf("the %d parameter of function %s is unspecified", i, typ.String())
		}
		argTypes = append(argTypes, param)
	}
	return argTypes, resultType, nil
}

func getFunctionTypeUniqueKey(name string, fun interface{}) string {
	k := reflect.TypeOf(fun).String()
	k = strings.ReplaceAll(k, " ", "_")
	return name + "_" + k
}

func wrapFunc(fun interface{}) (interface{}, error) {
	switch f := fun.(type) {
	case func(string) string:
		return callInStrOutStr(f), nil
	case func(string, string) bool:
		return callInStrStrOutBool(f), nil
	case func(string, string) (bool, error):
		return callInStrStrOutBoolWithErr(f), nil
	case func(bool, string, string) string:
		return callInBoolStrStrOutString(f), nil
	case func(string) net.IP:
		return callInStrOutIp(f), nil
	case func(string) (*mail.Address, error):
		return callInStrOutEmailWithErr(f), nil
	case func(string) (*url.URL, error):
		return callInStrOutURIWithErr(f), nil
	case func(string) (*DNSName, error):
		return callInStrOutDNSWithErr(f), nil
	case func() api.HeaderMap:
		return callOutStringMap(f), nil
	default:
		// This function can replace all the above cases, but it has a certain performance loss
		return reflectWrapFunc(fun)
	}
}

// reflectWrapFunc the safe wrapping of undefined wrapped function
func reflectWrapFunc(fun interface{}) (interface{}, error) {
	funVal := reflect.ValueOf(fun)
	if funVal.Kind() != reflect.Func {
		return nil, fmt.Errorf("must be a function, not a %s", funVal.Kind())
	}

	typ := funVal.Type()

	var needErr bool
	var result *expr.Type
	var resultKind attribute.Kind
	numOut := typ.NumOut()
	switch numOut {
	default:
		return nil, fmt.Errorf("too many result")
	case paraZero:
		return nil, fmt.Errorf("result is required")
	case paraTwo:
		if !typ.Out(1).AssignableTo(errType) {
			return nil, fmt.Errorf("last result must be error")
		}
		needErr = true
		fallthrough
	case paraOne:
		result = ConvertKind(typ.Out(0))
		if result == decls.Null {
			return nil, fmt.Errorf("the result of function %s is unspecified", typ.String())
		}
		resultKind = RecoverType(result)
		if resultKind == attribute.VALUE_TYPE_UNSPECIFIED {
			return nil, fmt.Errorf("the result of function %s is unspecified", typ.String())
		}
	}

	numIn := typ.NumIn()
	for i := 0; i != numIn; i++ {
		param := ConvertKind(typ.In(i))
		if param == decls.Null {
			return nil, fmt.Errorf("the %d parameter of function %s is unspecified", i, typ.String())
		}
	}

	funCall := func(values []reflect.Value) ref.Val {
		results := funVal.Call(values)
		if needErr && len(results) == 2 {
			err, _ := results[1].Interface().(error)
			if err != nil {
				return types.NewErr(err.Error())
			}
		}
		return ConvertValue(resultKind, results[0].Interface())
	}

	switch numIn {
	case paraOne:
		return functions.UnaryOp(func(value ref.Val) ref.Val {
			val, err := RecoverValue(value)
			if err != nil {
				return types.NewErr(err.Error())
			}
			return funCall([]reflect.Value{reflect.ValueOf(val)})
		}), nil
	case paraTwo:
		return functions.BinaryOp(func(lhs ref.Val, rhs ref.Val) ref.Val {
			lh, err := RecoverValue(lhs)
			if err != nil {
				return types.NewErr(err.Error())
			}
			rh, err := RecoverValue(rhs)
			if err != nil {
				return types.NewErr(err.Error())
			}
			return funCall([]reflect.Value{reflect.ValueOf(lh), reflect.ValueOf(rh)})
		}), nil
	case paraZero:
		return functions.FunctionOp(func(values ...ref.Val) ref.Val {
			return funCall([]reflect.Value{})
		}), nil
	default:
		return functions.FunctionOp(func(values ...ref.Val) ref.Val {
			vals := make([]reflect.Value, 0, len(values))
			for _, value := range values {
				val, err := RecoverValue(value)
				if err != nil {
					return types.NewErr(err.Error())
				}
				vals = append(vals, reflect.ValueOf(val))
			}
			return funCall(vals)
		}), nil
	}
}

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
		if len(args) != paraThree {
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

func callInStrOutIp(fn func(string) net.IP) functions.UnaryOp {
	return func(val ref.Val) ref.Val {
		arg1Val, ok := val.(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(val)
		}
		out := fn(string(arg1Val))

		return ipAddressValue{ip: out}
	}
}

func callInStrOutEmailWithErr(fn func(string) (*mail.Address, error)) functions.UnaryOp {
	return func(val ref.Val) ref.Val {
		arg1Val, ok := val.(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(val)
		}
		out, err := fn(string(arg1Val))
		if err != nil {
			return types.NewErr(err.Error())
		}
		return emailAddressValue{email: out}
	}
}

func callInStrOutURIWithErr(fn func(string) (*url.URL, error)) functions.UnaryOp {
	return func(val ref.Val) ref.Val {
		arg1Val, ok := val.(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(val)
		}
		out, err := fn(string(arg1Val))
		if err != nil {
			return types.NewErr(err.Error())
		}
		return uriValue{uri: out}
	}
}

func callInStrOutDNSWithErr(fn func(string) (*DNSName, error)) functions.UnaryOp {
	return func(val ref.Val) ref.Val {
		arg1Val, ok := val.(types.String)
		if !ok {
			return types.MaybeNoSuchOverloadErr(val)
		}
		out, err := fn(string(arg1Val))
		if err != nil {
			return types.NewErr(err.Error())
		}
		return dnsNameValue{dns: out}
	}
}

func callOutStringMap(fn func() api.HeaderMap) functions.FunctionOp {
	return func(values ...ref.Val) ref.Val {
		out := fn()
		return stringMapValue{headerMap: out}
	}
}
