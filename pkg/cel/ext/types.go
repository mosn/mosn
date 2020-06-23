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
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"
	expr "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"mosn.io/api"
	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/protocol"
)

func ConvertType(typ attribute.Kind) *expr.Type {
	switch typ {
	case attribute.STRING:
		return decls.String
	case attribute.INT64:
		return decls.Int
	case attribute.DOUBLE:
		return decls.Double
	case attribute.BOOL:
		return decls.Bool
	case attribute.TIMESTAMP:
		return decls.Timestamp
	case attribute.DURATION:
		return decls.Duration
	case attribute.STRING_MAP:
		return stringMapType
	case attribute.IP_ADDRESS:
		return decls.NewObjectType(ipAddressType)
	case attribute.EMAIL_ADDRESS:
		return decls.NewObjectType(emailAddressType)
	case attribute.URI:
		return decls.NewObjectType(uriType)
	case attribute.DNS_NAME:
		return decls.NewObjectType(dnsType)
	}
	return &expr.Type{TypeKind: &expr.Type_Dyn{}}
}

func RecoverType(typ *expr.Type) attribute.Kind {
	if typ == nil {
		return attribute.VALUE_TYPE_UNSPECIFIED
	}
	switch t := typ.TypeKind.(type) {
	case *expr.Type_Primitive:
		switch t.Primitive {
		case expr.Type_STRING:
			return attribute.STRING
		case expr.Type_INT64:
			return attribute.INT64
		case expr.Type_DOUBLE:
			return attribute.DOUBLE
		case expr.Type_BOOL:
			return attribute.BOOL
		}

	case *expr.Type_WellKnown:
		switch t.WellKnown {
		case expr.Type_TIMESTAMP:
			return attribute.TIMESTAMP
		case expr.Type_DURATION:
			return attribute.DURATION
		}

	case *expr.Type_MessageType:
		switch t.MessageType {
		case ipAddressType:
			return attribute.IP_ADDRESS
		case emailAddressType:
			return attribute.EMAIL_ADDRESS
		case uriType:
			return attribute.URI
		case dnsType:
			return attribute.DNS_NAME
		}

	case *expr.Type_MapType_:
		if reflect.DeepEqual(t.MapType.KeyType, decls.String) &&
			reflect.DeepEqual(t.MapType.ValueType, decls.String) {
			return attribute.STRING_MAP
		}

		// remaining maps are not yet supported
	}
	return attribute.VALUE_TYPE_UNSPECIFIED
}

func ConvertValue(typ attribute.Kind, value interface{}) ref.Val {
	switch typ {
	case attribute.STRING, attribute.INT64, attribute.DOUBLE, attribute.BOOL:
		return types.DefaultTypeAdapter.NativeToValue(value)
	case attribute.TIMESTAMP:
		t := value.(time.Time)
		tproto, err := ptypes.TimestampProto(t)
		if err != nil {
			return types.NewErr("incorrect timestamp: %v", err)
		}
		return types.Timestamp{Timestamp: tproto}
	case attribute.DURATION:
		d := value.(time.Duration)
		return types.Duration{Duration: ptypes.DurationProto(d)}
	case attribute.STRING_MAP:
		sm := value.(api.HeaderMap)
		return stringMapValue{headerMap: sm}
	case attribute.IP_ADDRESS:
		return wrapperValue{typ: typ, bytes: value.([]byte)}
	case attribute.EMAIL_ADDRESS, attribute.URI, attribute.DNS_NAME:
		return wrapperValue{typ: typ, s: value.(string)}
	}
	return types.NewErr("cannot convert value %#v of type %q", value, typ)
}

func RecoverValue(value ref.Val) (interface{}, error) {
	switch value.Type() {
	case types.ErrType:
		if err, ok := value.Value().(error); ok {
			return nil, err
		}
		return nil, errors.New("unrecoverable error value")
	case types.StringType, types.IntType, types.DoubleType, types.BoolType:
		return value.Value(), nil
	case types.TimestampType:
		t := value.Value().(*timestamp.Timestamp)
		return ptypes.Timestamp(t)
	case types.DurationType:
		d := value.Value().(*duration.Duration)
		return ptypes.Duration(d)
	case types.MapType:
		size := value.(traits.Sizer).Size()
		if size.Type() == types.IntType && size.Value().(int64) == 0 {
			return emptyStringMap.headerMap, nil
		}
		return value.Value(), nil
	case wrapperType:
		return value.Value(), nil
	case types.ListType:
		size := value.(traits.Sizer).Size()
		if size.Type() == types.IntType && size.Value().(int64) == 0 {
			return []string{}, nil
		}
		return value.Value(), nil
	}
	return nil, fmt.Errorf("failed to recover of type %s", value.Type())
}

var defaultValues = map[attribute.Kind]ref.Val{
	attribute.STRING:        types.String(""),
	attribute.INT64:         types.Int(0),
	attribute.DOUBLE:        types.Double(0),
	attribute.BOOL:          types.Bool(false),
	attribute.TIMESTAMP:     types.Timestamp{Timestamp: &timestamp.Timestamp{}},
	attribute.DURATION:      types.Duration{Duration: &duration.Duration{}},
	attribute.STRING_MAP:    emptyStringMap,
	attribute.IP_ADDRESS:    wrapperValue{typ: attribute.IP_ADDRESS, bytes: []byte{}},
	attribute.EMAIL_ADDRESS: wrapperValue{typ: attribute.EMAIL_ADDRESS, s: ""},
	attribute.URI:           wrapperValue{typ: attribute.URI, s: ""},
	attribute.DNS_NAME:      wrapperValue{typ: attribute.DNS_NAME, s: ""},
}

func DefaultValue(typ attribute.Kind) ref.Val {
	if out, ok := defaultValues[typ]; ok {
		return out
	}
	return types.NewErr("cannot provide defaults for %q", typ)
}

var (
	stringMapType = &expr.Type{TypeKind: &expr.Type_MapType_{MapType: &expr.Type_MapType{
		KeyType:   &expr.Type{TypeKind: &expr.Type_Primitive{Primitive: expr.Type_STRING}},
		ValueType: &expr.Type{TypeKind: &expr.Type_Primitive{Primitive: expr.Type_STRING}},
	}}}
	emptyStringMap = stringMapValue{headerMap: protocol.CommonHeader{}}

	// domain specific types do not implement any of type traits for now
	wrapperType = types.NewTypeValue("wrapper")
)

var (
	ipAddressType    = "IPAddress"
	emailAddressType = "EmailAddress"
	uriType          = "Uri"
	dnsType          = "DNSName"
)

type stringMapValue struct {
	headerMap api.HeaderMap
}

func (v stringMapValue) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	return nil, errors.New("cannot convert stringmap to native types")
}
func (v stringMapValue) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("cannot convert stringmap to CEL types")
}
func (v stringMapValue) Equal(other ref.Val) ref.Val {
	return types.NewErr("stringmap does not support equality")
}
func (v stringMapValue) Type() ref.Type {
	return types.MapType
}
func (v stringMapValue) Value() interface{} {
	return v.headerMap
}
func (v stringMapValue) Get(index ref.Val) ref.Val {
	if index.Type() != types.StringType {
		return types.NewErr("index should be a string")
	}

	field := index.Value().(string)
	value, found := v.headerMap.Get(field)
	if found {
		return types.String(value)
	}
	return types.NewErr("no such key: '%s'", field)
}
func (v stringMapValue) Contains(index ref.Val) ref.Val {
	if index.Type() != types.StringType {
		return types.NewErr("index should be a string")
	}

	field := index.Value().(string)
	_, found := v.headerMap.Get(field)
	return types.Bool(found)
}
func (v stringMapValue) Size() ref.Val {
	return types.NewErr("size not implemented on stringmaps")
}

type wrapperValue struct {
	typ   attribute.Kind
	bytes []byte
	s     string
}

func (v wrapperValue) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	return nil, errors.New("cannot convert wrapper value to native types")
}
func (v wrapperValue) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("cannot convert wrapper value  to CEL types")
}
func (v wrapperValue) Equal(other ref.Val) ref.Val {
	if other.Type() != wrapperType {
		return types.NewErr("cannot compare types")
	}
	w, ok := other.(wrapperValue)
	if !ok {
		return types.NewErr("cannot compare types")
	}
	if v.typ != w.typ {
		return types.NewErr("cannot compare %s and %s", v.typ, w.typ)
	}
	var out bool
	var err error
	switch v.typ {
	case attribute.IP_ADDRESS:
		out = externIPEqual(v.bytes, w.bytes)
	case attribute.DNS_NAME:
		out, err = externDNSNameEqual(v.s, w.s)
	case attribute.EMAIL_ADDRESS:
		out, err = externEmailEqual(v.s, w.s)
	case attribute.URI:
		out, err = externURIEqual(v.s, w.s)
	}
	if err != nil {
		return types.NewErr(err.Error())
	}
	return types.Bool(out)
}
func (v wrapperValue) Type() ref.Type {
	return wrapperType
}
func (v wrapperValue) Value() interface{} {
	switch v.typ {
	case attribute.IP_ADDRESS:
		return v.bytes
	default:
		return v.s
	}
}
