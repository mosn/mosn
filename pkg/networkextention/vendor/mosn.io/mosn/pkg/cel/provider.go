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
	"errors"
	"reflect"
	"strings"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	"github.com/google/cel-go/ext"
	"github.com/google/cel-go/interpreter"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"mosn.io/mosn/pkg/cel/attribute"
	cexlext "mosn.io/mosn/pkg/cel/ext"
)

// Attribute provider resolves typing information by modeling attributes
// as fields in nested proto messages with presence.
//
// For example, an integer attribute "a.b.c" is modeled as the following proto3 definition:
//
// message A {
//   message B {
//     google.protobuf.Int64Value c = 1;
//   }
//   B b = 1;
// }
//
// with root context containing a variable "a" of type "A".
//
// Synthetic message type names are dot-prepended attribute names, e.g. ".a.b.c".
type attributeProvider struct {
	// fallback proto-based type provider
	protos ref.TypeProvider

	root *node

	typeMap map[string]*node
}

// node corresponds to the message types holding other message types or scalars.
// leaf nodes have field types, inner nodes have children.
type node struct {
	typeName string

	children map[string]*node

	typ       *expr.Type
	valueType attribute.Kind
}

func (n *node) HasTrait(trait int) bool {
	return trait == traits.IndexerType || trait == traits.FieldTesterType
}
func (n *node) TypeName() string {
	return n.typeName
}

func (a *attributeProvider) newNode(typeName string) *node {
	out := &node{typeName: typeName}
	a.typeMap[typeName] = out
	return out
}

func (a *attributeProvider) insert(n *node, words []string, kind attribute.Kind) {
	if len(words) == 0 {
		n.typ = cexlext.ConvertType(kind)
		n.valueType = kind
		return
	}

	if n.children == nil {
		n.children = map[string]*node{}
	}

	child, ok := n.children[words[0]]
	if !ok {
		child = a.newNode(n.typeName + "." + words[0])
		n.children[words[0]] = child
	}

	a.insert(child, words[1:], kind)
}

func newAttributeProvider(attributes map[string]attribute.Kind) *attributeProvider {
	out := &attributeProvider{
		protos:  types.NewRegistry(),
		typeMap: map[string]*node{},
	}
	out.root = out.newNode("")
	for name, info := range attributes {
		out.insert(out.root, strings.Split(name, "."), info)
	}
	return out
}

func (a *attributeProvider) newEnvironment() *cel.Env {
	var declarations []*expr.Decl

	// populate with root-level identifiers
	for name, node := range a.root.children {
		if node.typ != nil {
			declarations = append(declarations, decls.NewVar(name, node.typ))
		} else {
			declarations = append(declarations, decls.NewVar(name, decls.NewObjectType(node.typeName)))
		}
	}

	// populate with standard functions
	// error is never expected here
	env, _ := cel.NewEnv(
		cel.CustomTypeProvider(a),
		cel.Declarations(declarations...),
		cexlext.StandardFunctionsEnvOption(),
		ext.Strings(),
		cexlext.Macros)

	return env
}

func (a *attributeProvider) newActivation(bag attribute.Bag) interpreter.Activation {
	return attributeActivation{provider: a, bag: bag}
}

func (a *attributeProvider) EnumValue(enumName string) ref.Val {
	return a.protos.EnumValue(enumName)
}
func (a *attributeProvider) FindIdent(identName string) (ref.Val, bool) {
	return a.protos.FindIdent(identName)
}
func (a *attributeProvider) FindType(typeName string) (*expr.Type, bool) {
	if _, ok := a.typeMap[typeName]; ok {
		return decls.NewObjectType(typeName), true
	}
	return a.protos.FindType(typeName)
}
func (a *attributeProvider) FindFieldType(messageName, fieldName string) (*ref.FieldType, bool) {
	node, ok := a.typeMap[messageName]
	if !ok {
		return a.protos.FindFieldType(messageName, fieldName)
	}

	child, ok := node.children[fieldName]
	if !ok {
		return nil, false
	}

	typ := child.typ
	if typ == nil {
		typ = decls.NewObjectType(child.typeName)
	}

	return &ref.FieldType{
		Type: typ,
	}, true
}
func (a *attributeProvider) NewValue(typeName string, fields map[string]ref.Val) ref.Val {
	return a.protos.NewValue(typeName, fields)
}

// Attribute activation binds attribute values to the expression nodes
type attributeActivation struct {
	provider *attributeProvider
	bag      attribute.Bag
}

type value struct {
	node *node
	bag  attribute.Bag
}

func (v value) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	return nil, errors.New("cannot convert attribute message to native types")
}
func (v value) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("cannot convert attribute message to CEL types")
}
func (v value) Equal(other ref.Val) ref.Val {
	return types.NewErr("attribute message does not support equality")
}
func (v value) Type() ref.Type {
	return v.node
}
func (v value) Value() interface{} {
	return v
}

func resolve(n *node, bag attribute.Bag) ref.Val {
	if n.typ == nil {
		return value{node: n, bag: bag}
	}
	value, found := bag.Get(n.typeName[1:])
	if found {
		return cexlext.ConvertValue(cexlext.RecoverType(n.typ), value)
	}
	return cexlext.DefaultValue(cexlext.RecoverType(n.typ))
}

func (v value) Get(index ref.Val) ref.Val {
	if index.Type() != types.StringType {
		return types.NewErr("select not implemented")
	}

	field := index.Value().(string)
	child, ok := v.node.children[field]
	if !ok {
		return types.NewErr("cannot evaluate select of %q from %s", field, v.node.typeName)
	}

	return resolve(child, v.bag)
}

func (v value) IsSet(index ref.Val) ref.Val {
	if index.Type() != types.StringType {
		return types.NewErr("select tester not implemented")
	}
	field := index.Value().(string)
	child, ok := v.node.children[field]
	if !ok {
		return types.NewErr("cannot evaluate select of %q from %s", field, v.node.typeName)
	}

	if child.typ != nil {
		_, found := v.bag.Get(child.typeName[1:])
		return types.Bool(found)
	}

	// assume all intermediate nodes are set
	return types.True
}

func (a attributeActivation) ResolveName(name string) (interface{}, bool) {
	if node, ok := a.provider.root.children[name]; ok {
		return resolve(node, a.bag), true
	}
	return nil, false
}

func (a attributeActivation) Parent() interpreter.Activation {
	return nil
}
