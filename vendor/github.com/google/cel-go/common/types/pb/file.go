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

	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
)

// NewFileDescription returns a FileDescription instance with a complete listing of all the message
// types and enum values declared within any scope in the file.
func NewFileDescription(fileDesc *descpb.FileDescriptorProto, pbdb *Db) *FileDescription {
	isProto3 := fileDesc.GetSyntax() == "proto3"
	metadata := collectFileMetadata(fileDesc)
	enums := make(map[string]*EnumValueDescription)
	for name, enumVal := range metadata.enumValues {
		enums[name] = NewEnumValueDescription(name, enumVal)
	}
	types := make(map[string]*TypeDescription)
	for name, msgType := range metadata.msgTypes {
		types[name] = NewTypeDescription(name, msgType, isProto3, pbdb.DescribeType)
	}
	return &FileDescription{
		types: types,
		enums: enums,
	}
}

// FileDescription holds a map of all types and enum values declared within a proto file.
type FileDescription struct {
	types map[string]*TypeDescription
	enums map[string]*EnumValueDescription
}

// GetEnumDescription returns an EnumDescription for a qualified enum value
// name declared within the .proto file.
func (fd *FileDescription) GetEnumDescription(enumName string) (*EnumValueDescription, error) {
	if ed, found := fd.enums[sanitizeProtoName(enumName)]; found {
		return ed, nil
	}
	return nil, fmt.Errorf("no such enum value '%s'", enumName)
}

// GetEnumNames returns the string names of all enum values in the file.
func (fd *FileDescription) GetEnumNames() []string {
	enumNames := make([]string, len(fd.enums))
	i := 0
	for _, e := range fd.enums {
		enumNames[i] = e.Name()
		i++
	}
	return enumNames
}

// GetTypeDescription returns a TypeDescription for a qualified type name
// declared within the .proto file.
func (fd *FileDescription) GetTypeDescription(typeName string) (*TypeDescription, error) {
	if td, found := fd.types[sanitizeProtoName(typeName)]; found {
		return td, nil
	}
	return nil, fmt.Errorf("no such type '%s'", typeName)
}

// GetTypeNames returns the list of all type names contained within the file.
func (fd *FileDescription) GetTypeNames() []string {
	typeNames := make([]string, len(fd.types))
	i := 0
	for _, t := range fd.types {
		typeNames[i] = t.Name()
		i++
	}
	return typeNames
}

// sanitizeProtoName strips the leading '.' from the proto message name.
func sanitizeProtoName(name string) string {
	if name != "" && name[0] == '.' {
		return name[1:]
	}
	return name
}

// fileMetadata is a flattened view of message types and enum values within a file descriptor.
type fileMetadata struct {
	// msgTypes maps from fully-qualified message name to descriptor.
	msgTypes map[string]*descpb.DescriptorProto
	// enumValues maps from fully-qualified enum value to enum value descriptor.
	enumValues map[string]*descpb.EnumValueDescriptorProto
}

// collectFileMetadata traverses the proto file object graph to collect message types and enum
// values and index them by their fully qualified names.
func collectFileMetadata(fileDesc *descpb.FileDescriptorProto) *fileMetadata {
	pkg := fileDesc.GetPackage()
	msgTypes := make(map[string]*descpb.DescriptorProto)
	collectMsgTypes(pkg, fileDesc.GetMessageType(), msgTypes)
	enumValues := make(map[string]*descpb.EnumValueDescriptorProto)
	collectEnumValues(pkg, fileDesc.GetEnumType(), enumValues)
	for container, msgType := range msgTypes {
		nestedEnums := msgType.GetEnumType()
		if len(nestedEnums) == 0 {
			continue
		}
		collectEnumValues(container, nestedEnums, enumValues)
	}
	return &fileMetadata{
		msgTypes:   msgTypes,
		enumValues: enumValues,
	}
}

// collectMsgTypes recursively collects messages and nested messages into a map of fully
// qualified message names to message descriptors.
func collectMsgTypes(container string,
	msgTypes []*descpb.DescriptorProto,
	msgTypeMap map[string]*descpb.DescriptorProto) {
	for _, msgType := range msgTypes {
		msgName := fmt.Sprintf("%s.%s", container, msgType.GetName())
		msgTypeMap[msgName] = msgType
		nestedTypes := msgType.GetNestedType()
		if len(nestedTypes) == 0 {
			continue
		}
		collectMsgTypes(msgName, nestedTypes, msgTypeMap)
	}
}

// collectEnumValues accumulates the enum values within an enum declaration.
func collectEnumValues(container string,
	enumTypes []*descpb.EnumDescriptorProto,
	enumValueMap map[string]*descpb.EnumValueDescriptorProto) {
	for _, enumType := range enumTypes {
		for _, enumValue := range enumType.GetValue() {
			name := fmt.Sprintf("%s.%s.%s", container, enumType.GetName(), enumValue.GetName())
			enumValueMap[name] = enumValue
		}
	}
}
