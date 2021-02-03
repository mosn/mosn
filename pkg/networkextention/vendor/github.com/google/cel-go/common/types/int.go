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
	"strconv"
	"time"

	"github.com/golang/protobuf/ptypes"

	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/common/types/traits"

	structpb "github.com/golang/protobuf/ptypes/struct"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
)

// Int type that implements ref.Val as well as comparison and math operators.
type Int int64

// Int constants used for comparison results.
const (
	// IntZero is the zero-value for Int
	IntZero   = Int(0)
	IntOne    = Int(1)
	IntNegOne = Int(-1)
)

var (
	// IntType singleton.
	IntType = NewTypeValue("int",
		traits.AdderType,
		traits.ComparerType,
		traits.DividerType,
		traits.ModderType,
		traits.MultiplierType,
		traits.NegatorType,
		traits.SubtractorType)

	// int32WrapperType reflected type for protobuf int32 wrapper type.
	int32WrapperType = reflect.TypeOf(&wrapperspb.Int32Value{})

	// int64WrapperType reflected type for protobuf int64 wrapper type.
	int64WrapperType = reflect.TypeOf(&wrapperspb.Int64Value{})
)

// Add implements traits.Adder.Add.
func (i Int) Add(other ref.Val) ref.Val {
	otherInt, ok := other.(Int)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	return i + otherInt
}

// Compare implements traits.Comparer.Compare.
func (i Int) Compare(other ref.Val) ref.Val {
	otherInt, ok := other.(Int)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	if i < otherInt {
		return IntNegOne
	}
	if i > otherInt {
		return IntOne
	}
	return IntZero
}

// ConvertToNative implements ref.Val.ConvertToNative.
func (i Int) ConvertToNative(typeDesc reflect.Type) (interface{}, error) {
	switch typeDesc.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64:
		// Enums are also mapped as int32 derivations.
		// Note, the code doesn't convert to the enum value directly since this is not known, but
		// the net effect with respect to proto-assignment is handled correctly by the reflection
		// Convert method.
		return reflect.ValueOf(i).Convert(typeDesc).Interface(), nil
	case reflect.Ptr:
		switch typeDesc {
		case anyValueType:
			// Primitives must be wrapped before being set on an Any field.
			return ptypes.MarshalAny(&wrapperspb.Int64Value{Value: int64(i)})
		case int32WrapperType:
			// Convert the value to a protobuf.Int32Value (with truncation).
			return &wrapperspb.Int32Value{Value: int32(i)}, nil
		case int64WrapperType:
			// Convert the value to a protobuf.Int64Value.
			return &wrapperspb.Int64Value{Value: int64(i)}, nil
		case jsonValueType:
			// The proto-to-JSON conversion rules would convert all 64-bit integer values to JSON
			// decimal strings. Because CEL ints might come from the automatic widening of 32-bit
			// values in protos, the JSON type is chosen dynamically based on the value.
			//
			// - Integers -2^53-1 < n < 2^53-1 are encoded as JSON numbers.
			// - Integers outside this range are encoded as JSON strings.
			//
			// The integer to float range represents the largest interval where such a conversion
			// can round-trip accurately. Thus, conversions from a 32-bit source can expect a JSON
			// number as with protobuf. Those consuming JSON from a 64-bit source must be able to
			// handle either a JSON number or a JSON decimal string. To handle these cases safely
			// the string values must be explicitly converted to int() within a CEL expression;
			// however, it is best to simply stay within the JSON number range when building JSON
			// objects in CEL.
			if i.isJSONSafe() {
				return &structpb.Value{
					Kind: &structpb.Value_NumberValue{NumberValue: float64(i)},
				}, nil
			}
			// Proto3 to JSON conversion requires string-formatted int64 values
			// since the conversion to floating point would result in truncation.
			return &structpb.Value{
				Kind: &structpb.Value_StringValue{
					StringValue: strconv.FormatInt(int64(i), 10),
				},
			}, nil
		}
		switch typeDesc.Elem().Kind() {
		case reflect.Int32:
			v := int32(i)
			p := reflect.New(typeDesc.Elem())
			p.Elem().Set(reflect.ValueOf(v).Convert(typeDesc.Elem()))
			return p.Interface(), nil
		case reflect.Int64:
			v := int64(i)
			p := reflect.New(typeDesc.Elem())
			p.Elem().Set(reflect.ValueOf(v).Convert(typeDesc.Elem()))
			return p.Interface(), nil
		}
	case reflect.Interface:
		if reflect.TypeOf(i).Implements(typeDesc) {
			return i, nil
		}
	}
	return nil, fmt.Errorf("unsupported type conversion from 'int' to %v", typeDesc)
}

// ConvertToType implements ref.Val.ConvertToType.
func (i Int) ConvertToType(typeVal ref.Type) ref.Val {
	switch typeVal {
	case IntType:
		return i
	case UintType:
		if i < 0 {
			return NewErr("range error converting %d to uint", i)
		}
		return Uint(i)
	case DoubleType:
		return Double(i)
	case StringType:
		return String(fmt.Sprintf("%d", int64(i)))
	case TimestampType:
		t := time.Unix(int64(i), 0)
		ts, err := ptypes.TimestampProto(t)
		if err != nil {
			return NewErr(err.Error())
		}
		return Timestamp{Timestamp: ts}
	case TypeType:
		return IntType
	}
	return NewErr("type conversion error from '%s' to '%s'", IntType, typeVal)
}

// Divide implements traits.Divider.Divide.
func (i Int) Divide(other ref.Val) ref.Val {
	otherInt, ok := other.(Int)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	if otherInt == IntZero {
		return NewErr("divide by zero")
	}
	return i / otherInt
}

// Equal implements ref.Val.Equal.
func (i Int) Equal(other ref.Val) ref.Val {
	otherInt, ok := other.(Int)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	return Bool(i == otherInt)
}

// Modulo implements traits.Modder.Modulo.
func (i Int) Modulo(other ref.Val) ref.Val {
	otherInt, ok := other.(Int)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	if otherInt == IntZero {
		return NewErr("modulus by zero")
	}
	return i % otherInt
}

// Multiply implements traits.Multiplier.Multiply.
func (i Int) Multiply(other ref.Val) ref.Val {
	otherInt, ok := other.(Int)
	if !ok {
		return ValOrErr(other, "no such overload")
	}
	return i * otherInt
}

// Negate implements traits.Negater.Negate.
func (i Int) Negate() ref.Val {
	return -i
}

// Subtract implements traits.Subtractor.Subtract.
func (i Int) Subtract(subtrahend ref.Val) ref.Val {
	subtraInt, ok := subtrahend.(Int)
	if !ok {
		return ValOrErr(subtrahend, "no such overload")
	}
	return i - subtraInt
}

// Type implements ref.Val.Type.
func (i Int) Type() ref.Type {
	return IntType
}

// Value implements ref.Val.Value.
func (i Int) Value() interface{} {
	return int64(i)
}

// isJSONSafe indicates whether the int is safely representable as a floating point value in JSON.
func (i Int) isJSONSafe() bool {
	return i >= minIntJSON && i <= maxIntJSON
}

const (
	// maxIntJSON is defined as the Number.MAX_SAFE_INTEGER value per EcmaScript 6.
	maxIntJSON = 1<<53 - 1
	// minIntJSON is defined as the Number.MIN_SAFE_INTEGER value per EcmaScript 6.
	minIntJSON = -maxIntJSON
)
