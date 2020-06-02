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

package hessian

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"
)

import (
	perrors "github.com/pkg/errors"
)

// invalid consts
const (
	InvalidJavaEnum JavaEnum = -1
)

// struct filed tag of hessian
var tagIdentifier = "hessian"

// SetTagIdentifier for customize struct filed tag of hessian, your can use it like:
//
// hessian.SetTagIdentifier("json")
//
// type MyUser struct {
// 	UserFullName      string   `json:"user_full_name"`
// 	FamilyPhoneNumber string   // default convert to => familyPhoneNumber
// }
//
// var user MyUser
// hessian.NewEncoder().Encode(user)
func SetTagIdentifier(s string) { tagIdentifier = s }

// POJO interface
// !!! Pls attention that Every field name should be upper case.
// Otherwise the app may panic.
type POJO interface {
	JavaClassName() string // got a go struct's Java Class package name which should be a POJO class.
}

// POJOEnum enum for POJO
type POJOEnum interface {
	POJO
	String() string
	EnumValue(string) JavaEnum
}

// JavaEnum type
type JavaEnum int32

// JavaEnumClass struct
type JavaEnumClass struct {
	name string
}

type classInfo struct {
	javaName      string
	fieldNameList []string
	buffer        []byte // encoded buffer
}

type structInfo struct {
	typ      reflect.Type
	goName   string
	javaName string
	index    int // classInfoList index
	inst     interface{}
}

// POJORegistry pojo registry struct
type POJORegistry struct {
	sync.RWMutex
	classInfoList []classInfo           // {class name, field name list...} list
	j2g           map[string]string     // java class name --> go struct name
	registry      map[string]structInfo // go class name --> go struct info
}

var (
	pojoRegistry = POJORegistry{
		j2g:      make(map[string]string),
		registry: make(map[string]structInfo),
	}
	pojoType     = reflect.TypeOf((*POJO)(nil)).Elem()
	javaEnumType = reflect.TypeOf((*POJOEnum)(nil)).Elem()
)

// struct parsing
func showPOJORegistry() {
	pojoRegistry.Lock()
	for k, v := range pojoRegistry.registry {
		fmt.Println("-->> show Registered types <<----")
		fmt.Println(k, v)
	}
	pojoRegistry.Unlock()
}

// RegisterPOJO Register a POJO instance. The return value is -1 if @o has been registered.
func RegisterPOJO(o POJO) int {
	// # definition for an object (compact map)
	// class-def  ::= 'C' string int string*
	pojoRegistry.Lock()
	defer pojoRegistry.Unlock()

	if goName, ok := pojoRegistry.j2g[o.JavaClassName()]; ok {
		return pojoRegistry.registry[goName].index
	}

	// JavaClassName shouldn't equal to goName
	if _, ok := pojoRegistry.registry[o.JavaClassName()]; ok {
		return -1
	}

	var (
		bHeader    []byte
		bBody      []byte
		fieldList  []string
		structInfo structInfo
		clsDef     classInfo
	)

	structInfo.typ = obtainValueType(o)

	structInfo.goName = structInfo.typ.String()
	structInfo.javaName = o.JavaClassName()
	structInfo.inst = o
	pojoRegistry.j2g[structInfo.javaName] = structInfo.goName
	registerTypeName(structInfo.goName, structInfo.javaName)

	// prepare fields info of objectDef
	nextStruct := []reflect.Type{structInfo.typ}
	for len(nextStruct) > 0 {
		current := nextStruct[0]

		for i := 0; i < current.NumField(); i++ {

			// skip unexported anonymous filed
			if current.Field(i).PkgPath != "" {
				continue
			}

			structField := current.Field(i)

			// skip ignored field
			tagVal, hasTag := structField.Tag.Lookup(tagIdentifier)
			if tagVal == `-` {
				continue
			}

			// flat anonymous field
			if structField.Anonymous && structField.Type.Kind() == reflect.Struct {
				nextStruct = append(nextStruct, structField.Type)
				continue
			}

			var fieldName string
			if hasTag {
				fieldName = tagVal
			} else {
				fieldName = lowerCamelCase(structField.Name)
			}

			fieldList = append(fieldList, fieldName)
			bBody = encString(bBody, fieldName)
		}

		nextStruct = nextStruct[1:]
	}

	// prepare header of objectDef
	bHeader = encByte(bHeader, BC_OBJECT_DEF)
	bHeader = encString(bHeader, structInfo.javaName)

	// write fields length into header of objectDef
	// note: cause fieldList is a dynamic slice, so one must calculate length only after it being prepared already.
	bHeader = encInt32(bHeader, int32(len(fieldList)))

	// prepare classDef
	clsDef = classInfo{javaName: structInfo.javaName, fieldNameList: fieldList}

	// merge header and body of objectDef into buffer of classInfo
	clsDef.buffer = append(bHeader, bBody...)

	structInfo.index = len(pojoRegistry.classInfoList)
	pojoRegistry.classInfoList = append(pojoRegistry.classInfoList, clsDef)
	pojoRegistry.registry[structInfo.goName] = structInfo

	return structInfo.index
}

// easy for test
func unRegisterPOJOs(os ...POJO) []int {
	arr := make([]int, len(os))
	for i := range os {
		arr[i] = unRegisterPOJO(os[i])
	}

	return arr
}

func unRegisterPOJO(o POJO) int {
	pojoRegistry.Lock()
	defer pojoRegistry.Unlock()

	goName := obtainValueType(o).String()

	if structInfo, ok := pojoRegistry.registry[goName]; ok {
		delete(pojoRegistry.j2g, structInfo.javaName)
		listTypeNameMapper.Delete(structInfo.goName)
		// remove registry cache.
		delete(pojoRegistry.registry, structInfo.goName)
		// don't remove registry classInfoList,
		// indexes of registered pojo may be affected.
		return structInfo.index
	}

	return -1
}

func obtainValueType(o POJO) reflect.Type {
	v := reflect.ValueOf(o)
	switch v.Kind() {
	case reflect.Struct:
		return v.Type()
	case reflect.Ptr:
		return v.Elem().Type()
	}

	return reflect.TypeOf(o)
}

// RegisterPOJOs register a POJO instance arr @os. The return value is @os's
// mathching index array, in which "-1" means its matching POJO has been registered.
func RegisterPOJOs(os ...POJO) []int {
	arr := make([]int, len(os))
	for i := range os {
		arr[i] = RegisterPOJO(os[i])
	}

	return arr
}

// RegisterJavaEnum Register a value type JavaEnum variable.
func RegisterJavaEnum(o POJOEnum) int {
	var (
		ok bool
		b  []byte
		i  int
		n  int
		f  string
		l  []string
		t  structInfo
		c  classInfo
		v  reflect.Value
	)

	pojoRegistry.Lock()
	defer pojoRegistry.Unlock()
	if _, ok = pojoRegistry.registry[o.JavaClassName()]; !ok {
		v = reflect.ValueOf(o)
		switch v.Kind() {
		case reflect.Struct:
			t.typ = v.Type()
		case reflect.Ptr:
			t.typ = v.Elem().Type()
		default:
			t.typ = reflect.TypeOf(o)
		}
		t.goName = t.typ.String()
		t.javaName = o.JavaClassName()
		t.inst = o
		pojoRegistry.j2g[t.javaName] = t.goName

		b = b[:0]
		b = encByte(b, BC_OBJECT_DEF)
		b = encString(b, t.javaName)
		l = l[:0]
		n = 1
		b = encInt32(b, int32(n))
		f = strings.ToLower("name")
		l = append(l, f)
		b = encString(b, f)

		c = classInfo{javaName: t.javaName, fieldNameList: l}
		c.buffer = append(c.buffer, b[:]...)
		t.index = len(pojoRegistry.classInfoList)
		pojoRegistry.classInfoList = append(pojoRegistry.classInfoList, c)
		pojoRegistry.registry[t.goName] = t
		i = t.index
	} else {
		i = -1
	}

	return i
}

// check if go struct name @goName has been registered or not.
func checkPOJORegistry(goName string) (int, bool) {
	var (
		ok bool
		s  structInfo
	)
	pojoRegistry.RLock()
	s, ok = pojoRegistry.registry[goName]
	pojoRegistry.RUnlock()

	return s.index, ok
}

// @typeName is class's java name
func getStructInfo(javaName string) (structInfo, bool) {
	var (
		ok bool
		g  string
		s  structInfo
	)

	pojoRegistry.RLock()
	g, ok = pojoRegistry.j2g[javaName]
	if ok {
		s, ok = pojoRegistry.registry[g]
	}
	pojoRegistry.RUnlock()

	return s, ok
}

func getStructDefByIndex(idx int) (reflect.Type, classInfo, error) {
	var (
		ok      bool
		clsName string
		cls     classInfo
		s       structInfo
	)

	pojoRegistry.RLock()
	defer pojoRegistry.RUnlock()

	if len(pojoRegistry.classInfoList) <= idx || idx < 0 {
		return nil, cls, perrors.Errorf("illegal class index @idx %d", idx)
	}
	cls = pojoRegistry.classInfoList[idx]
	clsName, ok = pojoRegistry.j2g[cls.javaName]
	if !ok {
		return nil, cls, perrors.Errorf("can not find java type name %s in registry", cls.javaName)
	}
	s, ok = pojoRegistry.registry[clsName]
	if !ok {
		return nil, cls, perrors.Errorf("can not find go type name %s in registry", clsName)
	}

	return s.typ, cls, nil
}

// Create a new instance by its struct name is @goName.
// the return value is nil if @o has been registered.
func createInstance(goName string) interface{} {
	var (
		ok bool
		s  structInfo
	)

	pojoRegistry.RLock()
	s, ok = pojoRegistry.registry[goName]
	pojoRegistry.RUnlock()
	if !ok {
		return nil
	}

	return reflect.New(s.typ).Interface()
}

func lowerCamelCase(s string) string {
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
