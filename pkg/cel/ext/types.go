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
	"context"
	"errors"
	"net"
	"net/mail"
	"net/url"
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
)

func ConvertType(typ attribute.Kind) *expr.Type {
	switch typ {
	case attribute.MOSN_CTX:
		return mosnCtxValueTypeWrapper
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
		return ipAddressValueTypeWrapper
	case attribute.EMAIL_ADDRESS:
		return emailAddressValueTypeWrapper
	case attribute.URI:
		return uriValueTypeWrapper
	case attribute.DNS_NAME:
		return dnsNameValueTypeWrapper
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
		case mosnCtxType:
			return attribute.MOSN_CTX
		case ipAddressType:
			return attribute.IP_ADDRESS
		case emailAddressType:
			return attribute.EMAIL_ADDRESS
		case uriType:
			return attribute.URI
		case dnsNameType:
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
	case attribute.MOSN_CTX:
		return &mosnCtx{value.(context.Context)}
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
		switch t := value.(type) {
		case []byte:
			return ipAddressValue{ip: t}
		case net.IP:
			return ipAddressValue{ip: t}
		}
	case attribute.EMAIL_ADDRESS:
		switch t := value.(type) {
		case string:
			email, _ := mail.ParseAddress(t)
			return emailAddressValue{email: email}
		case *mail.Address:
			return emailAddressValue{email: t}
		}
	case attribute.URI:
		switch t := value.(type) {
		case string:
			uri, _ := url.Parse(t)
			return uriValue{uri: uri}
		case *url.URL:
			return uriValue{uri: t}
		}

	case attribute.DNS_NAME:
		switch t := value.(type) {
		case string:
			return dnsNameValue{dns: &DNSName{
				Name: t,
			}}
		case *DNSName:
			return dnsNameValue{dns: t}
		}
	}
	return types.NewErr("cannot convert value %#v of type %q", value, typ)
}

func RecoverValue(value ref.Val) (interface{}, error) {
	switch value.Type() {
	case mosnCtxValueType:
		return value.Value(), nil
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
			return externEmptyStringMap(), nil
		}
		return value.Value(), nil
	case types.ListType:
		size := value.(traits.Sizer).Size()
		if size.Type() == types.IntType && size.Value().(int64) == 0 {
			return []string{}, nil
		}
		return value.Value(), nil
	default:
		return value.Value(), nil
	}
}

var defaultValues = map[attribute.Kind]ref.Val{
	attribute.MOSN_CTX:      &mosnCtx{Ctx: context.Background()},
	attribute.STRING:        types.String(""),
	attribute.INT64:         types.Int(0),
	attribute.DOUBLE:        types.Double(0),
	attribute.BOOL:          types.Bool(false),
	attribute.TIMESTAMP:     types.Timestamp{Timestamp: &timestamp.Timestamp{}},
	attribute.DURATION:      types.Duration{Duration: &duration.Duration{}},
	attribute.STRING_MAP:    stringMapValue{headerMap: externEmptyStringMap()},
	attribute.IP_ADDRESS:    ipAddressValue{},
	attribute.EMAIL_ADDRESS: emailAddressValue{},
	attribute.URI:           uriValue{},
	attribute.DNS_NAME:      dnsNameValue{},
}

func DefaultValue(typ attribute.Kind) ref.Val {
	if out, ok := defaultValues[typ]; ok {
		return out
	}
	return types.NewErr("cannot provide defaults for %q", typ)
}

var (
	stringMapType = decls.NewMapType(decls.String, decls.String)
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

var (
	ipAddressType             = "IPAddress"
	ipAddressValueType        = types.NewTypeValue(ipAddressType)
	ipAddressValueTypeWrapper = decls.NewObjectType(ipAddressType)
)

type ipAddressValue struct {
	ip net.IP
}

func (v ipAddressValue) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	return nil, errors.New("cannot convert wrapper value to native types")
}
func (v ipAddressValue) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("cannot convert wrapper value  to CEL types")
}
func (v ipAddressValue) Equal(other ref.Val) ref.Val {
	if other.Type() != ipAddressValueType {
		return types.NewErr("cannot compare types")
	}
	w, ok := other.(ipAddressValue)
	if !ok {
		return types.NewErr("cannot compare types")
	}
	return types.Bool(externIPEqual(v.ip, w.ip))
}
func (v ipAddressValue) Type() ref.Type {
	return ipAddressValueType
}
func (v ipAddressValue) Value() interface{} {
	return v.ip
}

var (
	emailAddressType             = "EmailAddress"
	emailAddressValueType        = types.NewTypeValue(emailAddressType)
	emailAddressValueTypeWrapper = decls.NewObjectType(emailAddressType)
)

type emailAddressValue struct {
	email *mail.Address
}

func (v emailAddressValue) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	return nil, errors.New("cannot convert wrapper value to native types")
}
func (v emailAddressValue) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("cannot convert wrapper value  to CEL types")
}
func (v emailAddressValue) Equal(other ref.Val) ref.Val {
	if other.Type() != emailAddressValueType {
		return types.NewErr("cannot compare types")
	}
	w, ok := other.(emailAddressValue)
	if !ok {
		return types.NewErr("cannot compare types")
	}
	b, err := externEmailEqual(v.email, w.email)
	if err != nil {
		return types.NewErr(err.Error())
	}
	return types.Bool(b)
}
func (v emailAddressValue) Type() ref.Type {
	return emailAddressValueType
}
func (v emailAddressValue) Value() interface{} {
	return v.email
}

var (
	uriType             = "Uri"
	uriValueType        = types.NewTypeValue(uriType)
	uriValueTypeWrapper = decls.NewObjectType(uriType)
)

type uriValue struct {
	uri *url.URL
}

func (v uriValue) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	return nil, errors.New("cannot convert wrapper value to native types")
}
func (v uriValue) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("cannot convert wrapper value  to CEL types")
}
func (v uriValue) Equal(other ref.Val) ref.Val {
	if other.Type() != uriValueType {
		return types.NewErr("cannot compare types")
	}
	w, ok := other.(uriValue)
	if !ok {
		return types.NewErr("cannot compare types")
	}
	b, err := externURIEqual(v.uri, w.uri)
	if err != nil {
		return types.NewErr(err.Error())
	}
	return types.Bool(b)
}
func (v uriValue) Type() ref.Type {
	return uriValueType
}
func (v uriValue) Value() interface{} {
	return v.uri
}

type DNSName struct {
	Name string
}

var (
	dnsNameType             = "DNSName"
	dnsNameValueType        = types.NewTypeValue(dnsNameType)
	dnsNameValueTypeWrapper = decls.NewObjectType(dnsNameType)
)

type dnsNameValue struct {
	dns *DNSName
}

func (v dnsNameValue) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	return nil, errors.New("cannot convert wrapper value to native types")
}
func (v dnsNameValue) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("cannot convert wrapper value  to CEL types")
}
func (v dnsNameValue) Equal(other ref.Val) ref.Val {
	if other.Type() != dnsNameValueType {
		return types.NewErr("cannot compare types")
	}
	w, ok := other.(dnsNameValue)
	if !ok {
		return types.NewErr("cannot compare types")
	}
	b, err := externDNSNameEqual(v.dns.Name, w.dns.Name)
	if err != nil {
		return types.NewErr(err.Error())
	}
	return types.Bool(b)
}
func (v dnsNameValue) Type() ref.Type {
	return dnsNameValueType
}
func (v dnsNameValue) Value() interface{} {
	return v.dns
}

var (
	mosnCtxType             = "mosnctx"
	mosnCtxValueType        = types.NewTypeValue(mosnCtxType)
	mosnCtxValueTypeWrapper = decls.NewObjectType(mosnCtxType)
)

type mosnCtx struct {
	Ctx context.Context
}

func (mosnCtx) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	return nil, errors.New("cannot convert mosnCtx to native types")
}

func (mosnCtx) ConvertToType(typeValue ref.Type) ref.Val {
	return types.NewErr("cannot convert mosnCtx to CEL types")
}

func (v *mosnCtx) Equal(other ref.Val) ref.Val {
	if other.Type() != mosnCtxValueType {
		return types.NewErr("connot compare types")
	}

	w, ok := other.(*mosnCtx)
	if !ok {
		return types.NewErr("connot compare types")
	}

	return types.Bool(w == v)
}

func (v mosnCtx) Type() ref.Type {
	return mosnCtxValueType
}

func (v mosnCtx) Value() interface{} {
	return v.Ctx
}

func ConvertKind(v reflect.Type) *expr.Type {
	switch v {
	case mosnCtxWrapperValueType:
		return mosnCtxValueTypeWrapper
	case boolType:
		return decls.Bool
	case intType, int32Type, int64Type:
		return decls.Int
	case float32Type, float64Type:
		return decls.Double
	case stringType:
		return decls.String
	case headerMapType:
		return stringMapType
	case timestampType:
		return decls.Timestamp
	case durationType:
		return decls.Duration
	case stringMapValueType:
		return stringMapType
	case ipAddressWrapperValueType:
		return ipAddressValueTypeWrapper
	case emailAddressWrapperValueType:
		return emailAddressValueTypeWrapper
	case uriWrapperValueType:
		return uriValueTypeWrapper
	case dnsNameWrapperValueType:
		return dnsNameValueTypeWrapper
	}
	return decls.Null
}

var (
	mosnCtxWrapperValueType = func() reflect.Type {
		var r context.Context
		return reflect.TypeOf(&r).Elem()
	}()
	stringMapValueType = func() reflect.Type {
		var r stringMapValue
		return reflect.TypeOf(r)
	}()
	ipAddressWrapperValueType = func() reflect.Type {
		var r net.IP
		return reflect.TypeOf(r)
	}()
	emailAddressWrapperValueType = func() reflect.Type {
		var r *mail.Address
		return reflect.TypeOf(&r).Elem()
	}()
	uriWrapperValueType = func() reflect.Type {
		var r *url.URL
		return reflect.TypeOf(&r).Elem()
	}()
	dnsNameWrapperValueType = func() reflect.Type {
		var r *DNSName
		return reflect.TypeOf(&r).Elem()
	}()
	errType = func() reflect.Type {
		var r error
		return reflect.TypeOf(&r).Elem()
	}()
	headerMapType = func() reflect.Type {
		var r api.HeaderMap
		return reflect.TypeOf(&r).Elem()
	}()
	timestampType = func() reflect.Type {
		var r time.Time
		return reflect.TypeOf(r)
	}()
	durationType = func() reflect.Type {
		var r time.Duration
		return reflect.TypeOf(r)
	}()
	boolType = func() reflect.Type {
		var r bool
		return reflect.TypeOf(r)
	}()
	intType = func() reflect.Type {
		var r int
		return reflect.TypeOf(r)
	}()
	int32Type = func() reflect.Type {
		var r int32
		return reflect.TypeOf(r)
	}()
	int64Type = func() reflect.Type {
		var r int64
		return reflect.TypeOf(r)
	}()
	float32Type = func() reflect.Type {
		var r float32
		return reflect.TypeOf(r)
	}()
	float64Type = func() reflect.Type {
		var r float64
		return reflect.TypeOf(r)
	}()
	stringType = func() reflect.Type {
		var r string
		return reflect.TypeOf(r)
	}()
)
