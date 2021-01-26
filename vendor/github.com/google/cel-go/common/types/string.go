// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package types

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/google/cel-go/common/overloads"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"

	structpb "github.com/golang/protobuf/ptypes/struct"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
)

// String type implementation which supports addition, comparison, matching,
// and size functions.
type String string

var (
	// StringType singleton.
	StringType = NewTypeValue("string",
		traits.AdderType,
		traits.ComparerType,
		traits.MatcherType,
		traits.ReceiverType,
		traits.SizerType)

	stringOneArgOverloads = map[string]func(String, ref.Val) ref.Val{
		overloads.Contains:   stringContains,
		overloads.EndsWith:   stringEndsWith,
		overloads.StartsWith: stringStartsWith,
	}

	stringWrapperType = reflect.TypeOf(&wrapperspb.StringValue{})
)

// Add implements traits.Adder.Add.
func (s String) Add(other ref.Val) ref.Val {
	otherString, ok := other.(String)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	return s + otherString
}

// Compare implements traits.Comparer.Compare.
func (s String) Compare(other ref.Val) ref.Val {
	otherString, ok := other.(String)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	return Int(strings.Compare(s.Value().(string), otherString.Value().(string)))
}

// ConvertToNative implements ref.Val.ConvertToNative.
func (s String) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	switch typeDesc.Kind() {
	case reflect.String:
		return s.Value(), nil
	case reflect.Ptr:
		switch typeDesc {
		case anyValueType:
			// Primitives must be wrapped before being set on an Any field.
			return ptypes.MarshalAny(&wrapperspb.StringValue{Value: string(s)})
		case jsonValueType:
			// Convert to a protobuf representation of a JSON String.
			return &structpb.Value{
				Kind: &structpb.Value_StringValue{StringValue: string(s)},
			}, nil
		case stringWrapperType:
			// Convert to a protobuf.StringValue.
			return &wrapperspb.StringValue{Value: string(s)}, nil
		}
		if typeDesc.Elem().Kind() == reflect.String {
			p := s.Value().(string)
			return &p, nil
		}
	case reflect.Interface:
		if reflect.TypeOf(s).Implements(typeDesc) {
			return s, nil
		}
	}
	return nil, fmt.Errorf(
		"unsupported native conversion from string to '%v'", typeDesc)
}

// ConvertToType implements ref.Val.ConvertToType.
func (s String) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case IntType:
		if n, err := strconv.ParseInt(s.Value().(string), 10, 64); err == nil {
			return Int(n)
		}
	case UintType:
		if n, err := strconv.ParseUint(s.Value().(string), 10, 64); err == nil {
			return Uint(n)
		}
	case DoubleType:
		if n, err := strconv.ParseFloat(s.Value().(string), 64); err == nil {
			return Double(n)
		}
	case BoolType:
		if b, err := strconv.ParseBool(s.Value().(string)); err == nil {
			return Bool(b)
		}
	case BytesType:
		return Bytes(s)
	case DurationType:
		if d, err := time.ParseDuration(s.Value().(string)); err == nil {
			return Duration{ptypes.DurationProto(d)}
		}
	case TimestampType:
		if t, err := time.Parse(time.RFC3339, s.Value().(string)); err == nil {
			if ts, err := ptypes.TimestampProto(t); err == nil {
				return Timestamp{ts}
			}
		}
	case StringType:
		return s
	case TypeType:
		return StringType
	}
	return NewErr("type conversion error from '%s' to '%s'", StringType, typeVal)
}

// Equal implements ref.Val.Equal.
func (s String) Equal(other ref.Val) ref.Val {
	otherString, ok := other.(String)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	return Bool(s == otherString)
}

// Match implements traits.Matcher.Match.
func (s String) Match(pattern ref.Val) ref.Val {
	pat, ok := pattern.(String)
	if !ok {
		return ValOrErr(pattern, "no such overload")
	}
	matched, err := regexp.MatchString(pat.Value().(string), s.Value().(string))
	if err != nil {
		return &Err{err}
	}
	return Bool(matched)
}

// Receive implements traits.Reciever.Receive.
func (s String) Receive(function string, overload string, args []ref.Val) ref.Val {
	switch len(args) {
	case 1:
		if f, found := stringOneArgOverloads[function]; found {
			return f(s, args[0])
		}
	}
	return NewErr("no such overload")
}

// Size implements traits.Sizer.Size.
func (s String) Size() ref.Val {
	return Int(len([]rune(s.Value().(string))))
}

// Type implements ref.Val.Type.
func (s String) Type() ref.Type {
	return StringType
}

// Value implements ref.Val.Value.
func (s String) Value() interface{} {
	return string(s)
}

func stringContains(s String, sub ref.Val) ref.Val {
	subStr, ok := sub.(String)
	if !ok {
		return ValOrErr(sub, "no such overload")
	}
	return Bool(strings.Contains(string(s), string(subStr)))
}

func stringEndsWith(s String, suf ref.Val) ref.Val {
	sufStr, ok := suf.(String)
	if !ok {
		return ValOrErr(suf, "no such overload")
	}
	return Bool(strings.HasSuffix(string(s), string(sufStr)))
}

func stringStartsWith(s String, pre ref.Val) ref.Val {
	preStr, ok := pre.(String)
	if !ok {
		return ValOrErr(pre, "no such overload")
	}
	return Bool(strings.HasPrefix(string(s), string(preStr)))
}
