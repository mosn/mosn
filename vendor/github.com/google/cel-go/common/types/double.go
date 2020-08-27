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
	"math"
	"reflect"

	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"

	"github.com/golang/protobuf/ptypes"

	structpb "github.com/golang/protobuf/ptypes/struct"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
)

// Double type that implements ref.Val, comparison, and mathematical
// operations.
type Double float64

var (
	// DoubleType singleton.
	DoubleType = NewTypeValue("double",
		traits.AdderType,
		traits.ComparerType,
		traits.DividerType,
		traits.MultiplierType,
		traits.NegatorType,
		traits.SubtractorType)

	// doubleWrapperType reflected type for protobuf double wrapper type.
	doubleWrapperType = reflect.TypeOf(&wrapperspb.DoubleValue{})

	// floatWrapperType reflected type for protobuf float wrapper type.
	floatWrapperType = reflect.TypeOf(&wrapperspb.FloatValue{})
)

// Add implements traits.Adder.Add.
func (d Double) Add(other ref.Val) ref.Val {
	otherDouble, ok := other.(Double)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	return d + otherDouble
}

// Compare implements traits.Comparer.Compare.
func (d Double) Compare(other ref.Val) ref.Val {
	otherDouble, ok := other.(Double)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	if d < otherDouble {
		return IntNegOne
	}
	if d > otherDouble {
		return IntOne
	}
	return IntZero
}

// ConvertToNative implements ref.Val.ConvertToNative.
func (d Double) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	switch typeDesc.Kind() {
	case reflect.Float32:
		v := float32(d)
		return reflect.ValueOf(v).Convert(typeDesc).Interface(), nil
	case reflect.Float64:
		v := float64(d)
		return reflect.ValueOf(v).Convert(typeDesc).Interface(), nil
	case reflect.Ptr:
		switch typeDesc {
		case anyValueType:
			// Primitives must be wrapped before being set on an Any field.
			return ptypes.MarshalAny(&wrapperspb.DoubleValue{Value: float64(d)})
		case doubleWrapperType:
			// Convert to a protobuf.DoubleValue
			return &wrapperspb.DoubleValue{Value: float64(d)}, nil
		case floatWrapperType:
			// Convert to a protobuf.FloatValue (with truncation).
			return &wrapperspb.FloatValue{Value: float32(d)}, nil
		case jsonValueType:
			// Note, there are special cases for proto3 to json conversion that
			// expect the floating point value to be converted to a NaN,
			// Infinity, or -Infinity string values, but the jsonpb string
			// marshaling of the protobuf.Value will handle this conversion.
			return &structpb.Value{
				Kind: &structpb.Value_NumberValue{NumberValue: float64(d)},
			}, nil
		}
		switch typeDesc.Elem().Kind() {
		case reflect.Float32:
			v := float32(d)
			p := reflect.New(typeDesc.Elem())
			p.Elem().Set(reflect.ValueOf(v).Convert(typeDesc.Elem()))
			return p.Interface(), nil
		case reflect.Float64:
			v := float64(d)
			p := reflect.New(typeDesc.Elem())
			p.Elem().Set(reflect.ValueOf(v).Convert(typeDesc.Elem()))
			return p.Interface(), nil
		}
	case reflect.Interface:
		if reflect.TypeOf(d).Implements(typeDesc) {
			return d, nil
		}
	}
	return nil, fmt.Errorf("type conversion error from Double to '%v'", typeDesc)
}

// ConvertToType implements ref.Val.ConvertToType.
func (d Double) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case IntType:
		i := math.Round(float64(d))
		if i > math.MaxInt64 || i < math.MinInt64 {
			return NewErr("range error converting %g to int", float64(d))
		}
		return Int(float64(i))
	case UintType:
		i := math.Round(float64(d))
		if i > math.MaxUint64 || i < 0 {
			return NewErr("range error converting %g to int", float64(d))
		}
		return Uint(float64(i))
	case DoubleType:
		return d
	case StringType:
		return String(fmt.Sprintf("%g", float64(d)))
	case TypeType:
		return DoubleType
	}
	return NewErr("type conversion error from '%s' to '%s'", DoubleType, typeVal)
}

// Divide implements traits.Divider.Divide.
func (d Double) Divide(other ref.Val) ref.Val {
	otherDouble, ok := other.(Double)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	return d / otherDouble
}

// Equal implements ref.Val.Equal.
func (d Double) Equal(other ref.Val) ref.Val {
	otherDouble, ok := other.(Double)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	// TODO: Handle NaNs properly.
	return Bool(d == otherDouble)
}

// Multiply implements traits.Multiplier.Multiply.
func (d Double) Multiply(other ref.Val) ref.Val {
	otherDouble, ok := other.(Double)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	return d * otherDouble
}

// Negate implements traits.Negater.Negate.
func (d Double) Negate() ref.Val {
	return -d
}

// Subtract implements traits.Subtractor.Subtract.
func (d Double) Subtract(subtrahend ref.Val) ref.Val {
	subtraDouble, ok := subtrahend.(Double)
	if !ok {
		return ValOrErr(subtrahend, "no such overload")
	}
	return d - subtraDouble
}

// Type implements ref.Val.Type.
func (d Double) Type() ref.Type {
	return DoubleType
}

// Value implements ref.Val.Value.
func (d Double) Value() interface{} {
	return float64(d)
}
