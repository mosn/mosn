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

// Package pb reflects over protocol buffer descriptors to generate objects
// that simplify type, enum, and field lookup.
package pb

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"

	"github.com/golang/protobuf/descriptor"
	"github.com/golang/protobuf/proto"

	descpb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	anypb "github.com/golang/protobuf/ptypes/any"
	durpb "github.com/golang/protobuf/ptypes/duration"
	structpb "github.com/golang/protobuf/ptypes/struct"
	tspb "github.com/golang/protobuf/ptypes/timestamp"
	wrapperspb "github.com/golang/protobuf/ptypes/wrappers"
)

// Db maps from file / message / enum name to file description.
type Db struct {
	revFileDescriptorMap map[string]*FileDescription
}

var (
	// DefaultDb used at evaluation time or unless overridden at check time.
	DefaultDb = &Db{
		revFileDescriptorMap: make(map[string]*FileDescription),
	}
)

// NewDb creates a new `pb.Db` with an empty type name to file description map.
func NewDb() *Db {
	pbdb := &Db{
		revFileDescriptorMap: make(map[string]*FileDescription),
	}
	// The FileDescription objects in the default db contain lazily initialized TypeDescription
	// values which may point to the state contained in the DefaultDb irrespective of this shallow
	// copy; however, the type graph for a field is idempotently computed, and is guaranteed to
	// only be initialized once thanks to atomic values within the TypeDescription objects, so it
	// is safe to share these values across instances.
	for k, v := range DefaultDb.revFileDescriptorMap {
		pbdb.revFileDescriptorMap[k] = v
	}
	return pbdb
}

// Copy creates a copy of the current database with its own internal descriptor mapping.
func (pbdb *Db) Copy() *Db {
	copy := NewDb()
	for k, v := range pbdb.revFileDescriptorMap {
		copy.revFileDescriptorMap[k] = v
	}
	return copy
}

// RegisterDescriptor produces a `FileDescription` from a `FileDescriptorProto` and registers the
// message and enum types into the `pb.Db`.
func (pbdb *Db) RegisterDescriptor(fileDesc *descpb.FileDescriptorProto) (*FileDescription, error) {
	fd, found := pbdb.revFileDescriptorMap[fileDesc.GetName()]
	if found {
		return fd, nil
	}
	fd = NewFileDescription(fileDesc, pbdb)
	for _, enumValName := range fd.GetEnumNames() {
		pbdb.revFileDescriptorMap[enumValName] = fd
	}
	for _, msgTypeName := range fd.GetTypeNames() {
		pbdb.revFileDescriptorMap[msgTypeName] = fd
	}
	pbdb.revFileDescriptorMap[fileDesc.GetName()] = fd

	// Return the specific file descriptor registered.
	return fd, nil
}

// RegisterMessage produces a `FileDescription` from a `message` and registers the message and all
// other definitions within the message file into the `pb.Db`.
func (pbdb *Db) RegisterMessage(message proto.Message) (*FileDescription, error) {
	typeName := sanitizeProtoName(proto.MessageName(message))
	if fd, found := pbdb.revFileDescriptorMap[typeName]; found {
		return fd, nil
	}
	fileDesc, _ := descriptor.ForMessage(message.(descriptor.Message))
	return pbdb.RegisterDescriptor(fileDesc)
}

// DescribeFile gets the `FileDescription` for the `message` type if it exists in the `pb.Db`.
func (pbdb *Db) DescribeFile(message proto.Message) (*FileDescription, error) {
	typeName := sanitizeProtoName(proto.MessageName(message))
	if fd, found := pbdb.revFileDescriptorMap[typeName]; found {
		return fd, nil
	}
	return nil, fmt.Errorf("unrecognized proto type name '%s'", typeName)
}

// DescribeEnum takes a qualified enum name and returns an `EnumDescription` if it exists in the
// `pb.Db`.
func (pbdb *Db) DescribeEnum(enumName string) (*EnumValueDescription, error) {
	enumName = sanitizeProtoName(enumName)
	if fd, found := pbdb.revFileDescriptorMap[enumName]; found {
		return fd.GetEnumDescription(enumName)
	}
	return nil, fmt.Errorf("unrecognized enum '%s'", enumName)
}

// DescribeType returns a `TypeDescription` for the `typeName` if it exists in the `pb.Db`.
func (pbdb *Db) DescribeType(typeName string) (*TypeDescription, error) {
	typeName = sanitizeProtoName(typeName)
	if fd, found := pbdb.revFileDescriptorMap[typeName]; found {
		return fd.GetTypeDescription(typeName)
	}
	return nil, fmt.Errorf("unrecognized type '%s'", typeName)
}

// CollectFileDescriptorSet builds a file descriptor set associated with the file where the input
// message is declared.
func CollectFileDescriptorSet(message proto.Message) (*descpb.FileDescriptorSet, error) {
	fdMap := map[string]*descpb.FileDescriptorProto{}
	fd, _ := descriptor.ForMessage(message.(descriptor.Message))
	fdMap[fd.GetName()] = fd
	// Initialize list of dependencies
	fileDeps := fd.GetDependency()
	deps := make([]string, len(fileDeps))
	copy(deps, fileDeps)
	// Expand list for new dependencies
	for i := 0; i < len(deps); i++ {
		dep := deps[i]
		if _, found := fdMap[dep]; found {
			continue
		}
		depDesc, err := readFileDescriptor(dep)
		if err != nil {
			return nil, err
		}
		fdMap[dep] = depDesc
		deps = append(deps, depDesc.GetDependency()...)
	}

	fds := make([]*descpb.FileDescriptorProto, len(fdMap), len(fdMap))
	i := 0
	for _, fd = range fdMap {
		fds[i] = fd
		i++
	}
	return &descpb.FileDescriptorSet{
		File: fds,
	}, nil
}

// readFileDescriptor will read the gzipped file descriptor for a given proto file and return the
// hydrated FileDescriptorProto.
//
// If the file name is not found or there is an error during deserialization an error is returned.
func readFileDescriptor(protoFileName string) (*descpb.FileDescriptorProto, error) {
	gzipped := proto.FileDescriptor(protoFileName)
	r, err := gzip.NewReader(bytes.NewReader(gzipped))
	if err != nil {
		return nil, fmt.Errorf("bad gzipped descriptor: %v", err)
	}
	unzipped, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("bad gzipped descriptor: %v", err)
	}
	fd := &descpb.FileDescriptorProto{}
	if err := proto.Unmarshal(unzipped, fd); err != nil {
		return nil, fmt.Errorf("bad gzipped descriptor: %v", err)
	}
	return fd, nil
}

func init() {
	// Describe well-known types to ensure they can always be resolved by the check and interpret
	// execution phases.
	//
	// The following subset of message types is enough to ensure that all well-known types can
	// resolved in the runtime, since describing the value results in describing the whole file
	// where the message is declared.
	DefaultDb.RegisterMessage(&anypb.Any{})
	DefaultDb.RegisterMessage(&durpb.Duration{})
	DefaultDb.RegisterMessage(&tspb.Timestamp{})
	DefaultDb.RegisterMessage(&structpb.Value{})
	DefaultDb.RegisterMessage(&wrapperspb.BoolValue{})
}
