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

package pb

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/golang/protobuf/proto"
	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	structpb "github.com/golang/protobuf/ptypes/struct"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

// NewTypeDescription produces a TypeDescription value for the fully-qualified proto type name
// with a given descriptor.
//
// The type description creation method also expects the type to be marked clearly as a proto2 or
// proto3 type, and accepts a typeResolver reference for resolving field TypeDescription during
// lazily initialization of the type which is done atomically.
func NewTypeDescription(typeName string, desc *descpb.DescriptorProto,
	isProto3 bool, resolveType typeResolver) *TypeDescription {
	return &TypeDescription{
		typeName:    typeName,
		isProto3:    isProto3,
		desc:        desc,
		resolveType: resolveType,
	}
}

// TypeDescription is a collection of type metadata relevant to expression
// checking and evaluation.
type TypeDescription struct {
	typeName string
	isProto3 bool
	desc     *descpb.DescriptorProto

	// resolveType is used to lookup field types during type initialization.
	// The resolver may point to shared state; however, this state is guaranteed to be computed at
	// most one time.
	resolveType typeResolver
	init        sync.Once
	metadata    *typeMetadata
}

// typeResolver accepts a type name and returns a TypeDescription.
// The typeResolver is used to resolve field types during lazily initialization of the type
// description metadata.
type typeResolver func(typeName string) (*TypeDescription, error)

type typeMetadata struct {
	fields          map[string]*FieldDescription // fields by name (proto)
	fieldIndices    map[int][]*FieldDescription  // fields by Go struct idx
	fieldProperties *proto.StructProperties
	reflectedType   *reflect.Type
	reflectedVal    *reflect.Value
	emptyVal        interface{}
}

// FieldCount returns the number of fields declared within the type.
func (td *TypeDescription) FieldCount() int {
	// The number of keys in the field indices map corresponds to the number
	// of fields on the proto message.
	return len(td.getMetadata().fieldIndices)
}

// FieldByName returns the FieldDescription associated with a field name.
func (td *TypeDescription) FieldByName(name string) (*FieldDescription, bool) {
	fd, found := td.getMetadata().fields[name]
	return fd, found
}

// Name of the type.
func (td *TypeDescription) Name() string {
	return td.typeName
}

// ReflectType returns the reflected struct type of the generated proto struct.
func (td *TypeDescription) ReflectType() reflect.Type {
	if td.getMetadata().reflectedType == nil {
		return nil
	}
	return *td.getMetadata().reflectedType
}

// DefaultValue returns an empty instance of the proto message associated with the type,
// or nil for wrapper types.
func (td *TypeDescription) DefaultValue() proto.Message {
	val := td.getMetadata().emptyVal
	if val == nil {
		return nil
	}
	return val.(proto.Message)
}

// getMetadata computes the type field metadata used for determining field types and default
// values. The call to makeMetadata within this method is guaranteed to be invoked exactly
// once.
func (td *TypeDescription) getMetadata() *typeMetadata {
	td.init.Do(func() {
		td.metadata = td.makeMetadata()
	})
	return td.metadata
}

func (td *TypeDescription) makeMetadata() *typeMetadata {
	refType := proto.MessageType(td.typeName)
	meta := &typeMetadata{
		fields:       make(map[string]*FieldDescription),
		fieldIndices: make(map[int][]*FieldDescription),
	}
	if refType != nil {
		// Set the reflected type if non-nil.
		meta.reflectedType = &refType

		// Unwrap the pointer reference for the sake of later checks.
		elemType := refType
		if elemType.Kind() == reflect.Ptr {
			elemType = elemType.Elem()
		}
		if elemType.Kind() == reflect.Struct {
			meta.fieldProperties = proto.GetProperties(elemType)
		}
		refVal := reflect.New(elemType)
		meta.reflectedVal = &refVal
		if refVal.CanInterface() {
			meta.emptyVal = refVal.Interface()
		} else {
			meta.emptyVal = reflect.Zero(elemType).Interface()
		}
	}

	fieldIndexMap := make(map[string]int)
	fieldDescMap := make(map[string]*descpb.FieldDescriptorProto)
	for i, f := range td.desc.Field {
		fieldDescMap[f.GetName()] = f
		fieldIndexMap[f.GetName()] = i
	}
	if meta.fieldProperties != nil {
		// This is a proper message type.
		for i, prop := range meta.fieldProperties.Prop {
			if strings.HasPrefix(prop.OrigName, "XXX_") {
				// Book-keeping fields generated by protoc start with XXX_
				continue
			}
			desc := fieldDescMap[prop.OrigName]
			fd := td.newFieldDesc(*meta.reflectedType, desc, prop, i)
			meta.fields[prop.OrigName] = fd
			meta.fieldIndices[i] = append(meta.fieldIndices[i], fd)
		}
		for _, oneofProp := range meta.fieldProperties.OneofTypes {
			desc := fieldDescMap[oneofProp.Prop.OrigName]
			fd := td.newOneofFieldDesc(*meta.reflectedType, desc, oneofProp, oneofProp.Field)
			meta.fields[oneofProp.Prop.OrigName] = fd
			meta.fieldIndices[oneofProp.Field] = append(meta.fieldIndices[oneofProp.Field], fd)
		}
	} else {
		for fieldName, desc := range fieldDescMap {
			fd := td.newMapFieldDesc(desc)
			meta.fields[fieldName] = fd
			index := fieldIndexMap[fieldName]
			meta.fieldIndices[index] = append(meta.fieldIndices[index], fd)
		}
	}
	return meta
}

// Create a new field description for the proto field descriptor associated with the given type.
// The field properties should never not be found when performing reflection on the type unless
// there are fundamental changes to the backing proto library behavior.
func (td *TypeDescription) newFieldDesc(
	tdType reflect.Type,
	desc *descpb.FieldDescriptorProto,
	prop *proto.Properties,
	index int) *FieldDescription {
	getterName := fmt.Sprintf("Get%s", prop.Name)
	getter, _ := tdType.MethodByName(getterName)
	var field *reflect.StructField
	if tdType.Kind() == reflect.Ptr {
		tdType = tdType.Elem()
	}
	f, found := tdType.FieldByName(prop.Name)
	if found {
		field = &f
	}
	fieldDesc := &FieldDescription{
		desc:      desc,
		index:     index,
		getter:    getter.Func,
		field:     field,
		prop:      prop,
		isProto3:  td.isProto3,
		isWrapper: isWrapperType(desc),
	}
	if desc.GetType() == descpb.FieldDescriptorProto_TYPE_MESSAGE {
		typeName := sanitizeProtoName(desc.GetTypeName())
		fieldType, _ := td.resolveType(typeName)
		fieldDesc.td = fieldType
		return fieldDesc
	}
	return fieldDesc
}

func (td *TypeDescription) newOneofFieldDesc(
	tdType reflect.Type,
	desc *descpb.FieldDescriptorProto,
	oneofProp *proto.OneofProperties,
	index int) *FieldDescription {
	fieldDesc := td.newFieldDesc(tdType, desc, oneofProp.Prop, index)
	fieldDesc.oneofProp = oneofProp
	return fieldDesc
}

func (td *TypeDescription) newMapFieldDesc(desc *descpb.FieldDescriptorProto) *FieldDescription {
	return &FieldDescription{
		desc:     desc,
		index:    int(desc.GetNumber()),
		isProto3: td.isProto3,
	}
}

func isWrapperType(desc *descpb.FieldDescriptorProto) bool {
	if desc.GetType() != descpb.FieldDescriptorProto_TYPE_MESSAGE {
		return false
	}
	switch sanitizeProtoName(desc.GetTypeName()) {
	case "google.protobuf.BoolValue",
		"google.protobuf.BytesValue",
		"google.protobuf.DoubleValue",
		"google.protobuf.FloatValue",
		"google.protobuf.Int32Value",
		"google.protobuf.Int64Value",
		"google.protobuf.StringValue",
		"google.protobuf.UInt32Value",
		"google.protobuf.UInt64Value":
		return true
	}
	return false
}

// FieldDescription holds metadata related to fields declared within a type.
type FieldDescription struct {
	// getter is the reflected accessor method that obtains the field value.
	getter reflect.Value
	// field is the field location in a refValue
	// The field will be not found for oneofs, but this is accounted for
	// by checking the 'desc' value which provides this information.
	field *reflect.StructField
	// isProto3 indicates whether the field is defined in a proto3 syntax.
	isProto3 bool
	// isWrapper indicates whether the field is a wrapper type.
	isWrapper bool

	// td is the type description for message typed fields.
	td *TypeDescription

	// proto descriptor data.
	desc      *descpb.FieldDescriptorProto
	index     int
	prop      *proto.Properties
	oneofProp *proto.OneofProperties
}

// CheckedType returns the type-definition used at type-check time.
func (fd *FieldDescription) CheckedType() *exprpb.Type {
	if fd.IsMap() {
		// Get the FieldDescriptors for the type arranged by their index within the
		// generated Go struct.
		fieldIndices := fd.getFieldIndicies()
		// Map keys and values are represented as repeated entries in a list.
		key := fieldIndices[0][0]
		val := fieldIndices[1][0]
		return &exprpb.Type{
			TypeKind: &exprpb.Type_MapType_{
				MapType: &exprpb.Type_MapType{
					KeyType:   key.typeDefToType(),
					ValueType: val.typeDefToType()}}}
	}
	if fd.IsRepeated() {
		return &exprpb.Type{
			TypeKind: &exprpb.Type_ListType_{
				ListType: &exprpb.Type_ListType{
					ElemType: fd.typeDefToType()}}}
	}
	return fd.typeDefToType()
}

// IsSet returns whether the field is set on the target value, per the proto presence conventions
// of proto2 or proto3 accordingly.
//
// The input target may either be a reflect.Value or Go struct type.
func (fd *FieldDescription) IsSet(target interface{}) bool {
	t, ok := target.(reflect.Value)
	if !ok {
		t = reflect.ValueOf(target)
	}
	// For the case where the field is not a oneof, test whether the field is set on the target
	// value assuming it is a struct. A field that is not set will be one of the following values:
	// - nil for message and primitive typed fields in proto2
	// - nil for message typed fields in proto3
	// - empty for primitive typed fields in proto3
	if fd.field != nil && !fd.IsOneof() {
		t = reflect.Indirect(t)
		return isFieldSet(t.FieldByIndex(fd.field.Index))
	}
	// Oneof fields must consider two pieces of information:
	// - whether the oneof is set to any value at all
	// - whether the field in the oneof is the same as the field under test.
	//
	// In go protobuf libraries, oneofs result in the creation of special oneof type messages
	// which contain a reference to the actual field type. The creation of these special message
	// types makes it possible to test for presence of primitive field values in proto3.
	//
	// The logic below performs a get on the oneof to obtain the field reference and then checks
	// the type of the field reference against the known oneof type determined FieldDescription
	// initialization.
	if fd.IsOneof() {
		t = reflect.Indirect(t)
		oneof := t.Field(fd.Index())
		if !isFieldSet(oneof) {
			return false
		}
		oneofVal := oneof.Interface()
		oneofType := reflect.TypeOf(oneofVal)
		return oneofType == fd.OneofType()
	}

	// When the field is nil or when the field is a oneof, call the accessor
	// associated with this field name to determine whether the field value is
	// the default.
	fieldVal := fd.getter.Call([]reflect.Value{t})[0]
	return isFieldSet(fieldVal)
}

// GetFrom returns the accessor method associated with the field on the proto generated struct.
//
// If the field is not set, the proto default value is returned instead.
//
// The input target may either be a reflect.Value or Go struct type.
func (fd *FieldDescription) GetFrom(target interface{}) (interface{}, error) {
	t, ok := target.(reflect.Value)
	if !ok {
		t = reflect.ValueOf(target)
	}
	var fieldVal reflect.Value
	if fd.isProto3 && fd.field != nil && !fd.IsOneof() {
		// The target object should always be a struct.
		t = reflect.Indirect(t)
		if t.Kind() != reflect.Struct {
			return nil, fmt.Errorf("unsupported field selection target: %T", target)
		}
		fieldVal = t.FieldByIndex(fd.field.Index)
	} else {
		// The accessor method must be used for proto2 in order to properly handle
		// default values.
		// Additionally, proto3 oneofs require the use of the accessor to get the proper value.
		fieldVal = fd.getter.Call([]reflect.Value{t})[0]
	}
	// If the field is a non-repeated message, and it's not set, return its default value.
	// Note, repeated fields should have default values of empty list or empty map, so the checks
	// for whether to return a default proto message don't really apply.
	if fd.IsMessage() && !fd.IsRepeated() && !isFieldSet(fieldVal) {
		// Well known wrapper types default to null if not set.
		if fd.IsWrapper() {
			return structpb.NullValue_NULL_VALUE, nil
		}
		// Otherwise, return an empty message.
		return fd.Type().DefaultValue(), nil
	}
	// Otherwise, return the field value or the zero value for its type.
	if fieldVal.CanInterface() {
		return fieldVal.Interface(), nil
	}
	return reflect.Zero(fieldVal.Type()).Interface(), nil
}

// Index returns the field index within a reflected value.
func (fd *FieldDescription) Index() int {
	return fd.index
}

// IsEnum returns true if the field type refers to an enum value.
func (fd *FieldDescription) IsEnum() bool {
	return fd.desc.GetType() == descpb.FieldDescriptorProto_TYPE_ENUM
}

// IsMap returns true if the field is of map type.
func (fd *FieldDescription) IsMap() bool {
	if !fd.IsRepeated() || !fd.IsMessage() {
		return false
	}
	if fd.td == nil {
		return false
	}
	return fd.td.desc.GetOptions().GetMapEntry()
}

// IsMessage returns true if the field is of message type.
func (fd *FieldDescription) IsMessage() bool {
	return fd.desc.GetType() == descpb.FieldDescriptorProto_TYPE_MESSAGE
}

// IsOneof returns true if the field is declared within a oneof block.
func (fd *FieldDescription) IsOneof() bool {
	if fd.desc != nil {
		return fd.desc.OneofIndex != nil
	}
	return fd.oneofProp != nil
}

// IsRepeated returns true if the field is a repeated value.
//
// This method will also return true for map values, so check whether the
// field is also a map.
func (fd *FieldDescription) IsRepeated() bool {
	return *fd.desc.Label == descpb.FieldDescriptorProto_LABEL_REPEATED
}

// IsWrapper returns true if the field type is a primitive wrapper type.
func (fd *FieldDescription) IsWrapper() bool {
	return fd.isWrapper
}

// OneofType returns the reflect.Type value of a oneof field.
//
// Oneof field values are wrapped in a struct which contains one field whose
// value is a proto.Message.
func (fd *FieldDescription) OneofType() reflect.Type {
	return fd.oneofProp.Type
}

// OrigName returns the snake_case name of the field as it was declared within
// the proto. This is the same name format that is expected within expressions.
func (fd *FieldDescription) OrigName() string {
	if fd.desc != nil && fd.desc.Name != nil {
		return *fd.desc.Name
	}
	return fd.prop.OrigName
}

// Name returns the CamelCase name of the field within the proto-based struct.
func (fd *FieldDescription) Name() string {
	return fd.prop.Name
}

// String returns a struct-like field definition string.
func (fd *FieldDescription) String() string {
	return fmt.Sprintf("%s %s `oneof=%t`",
		fd.TypeName(), fd.OrigName(), fd.IsOneof())
}

// Type returns the TypeDescription for the field.
func (fd *FieldDescription) Type() *TypeDescription {
	return fd.td
}

// TypeName returns the type name of the field.
func (fd *FieldDescription) TypeName() string {
	return sanitizeProtoName(fd.desc.GetTypeName())
}

func (fd *FieldDescription) getFieldIndicies() map[int][]*FieldDescription {
	return fd.td.getMetadata().fieldIndices
}

func (fd *FieldDescription) typeDefToType() *exprpb.Type {
	if fd.IsMessage() {
		if wk, found := CheckedWellKnowns[fd.TypeName()]; found {
			return wk
		}
		return checkedMessageType(fd.TypeName())
	}
	if fd.IsEnum() {
		return checkedInt
	}
	if p, found := CheckedPrimitives[fd.desc.GetType()]; found {
		return p
	}
	return CheckedPrimitives[fd.desc.GetType()]
}

func checkedMessageType(name string) *exprpb.Type {
	return &exprpb.Type{
		TypeKind: &exprpb.Type_MessageType{MessageType: name}}
}

func checkedPrimitive(primitive exprpb.Type_PrimitiveType) *exprpb.Type {
	return &exprpb.Type{
		TypeKind: &exprpb.Type_Primitive{Primitive: primitive}}
}

func checkedWellKnown(wellKnown exprpb.Type_WellKnownType) *exprpb.Type {
	return &exprpb.Type{
		TypeKind: &exprpb.Type_WellKnown{WellKnown: wellKnown}}
}

func checkedWrap(t *exprpb.Type) *exprpb.Type {
	return &exprpb.Type{
		TypeKind: &exprpb.Type_Wrapper{Wrapper: t.GetPrimitive()}}
}

func isFieldSet(refVal reflect.Value) bool {
	switch refVal.Kind() {
	case reflect.Ptr:
		// proto2 represents all non-repeated fields as pointers.
		// proto3 represents message fields as pointers.
		// if the value is non-nil, it is set.
		return !refVal.IsNil()
	case reflect.Array, reflect.Slice, reflect.Map:
		// proto2 and proto3 repeated and map types are considered set if not empty.
		return refVal.Len() > 0
	default:
		// proto3 represents simple types by their zero value when they are not set.
		// return whether the value is something other than the zero value.
		zeroVal := reflect.Zero(refVal.Type()).Interface()
		if refVal.CanInterface() {
			val := refVal.Interface()
			return !reflect.DeepEqual(val, zeroVal)
		}
		return false
	}
}
