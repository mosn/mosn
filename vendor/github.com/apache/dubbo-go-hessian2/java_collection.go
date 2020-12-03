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
	"reflect"
)

import (
	perrors "github.com/pkg/errors"
)

type JavaCollectionObject interface {
	Get() []interface{}
	Set([]interface{})
	JavaClassName() string
}

var collectionTypeMap = make(map[string]reflect.Type, 16)

func SetCollectionSerialize(collection JavaCollectionObject) {
	name := collection.JavaClassName()
	v := reflect.ValueOf(collection)
	var typ reflect.Type
	switch v.Kind() {
	case reflect.Struct:
		typ = v.Type()
	case reflect.Ptr:
		typ = v.Elem().Type()
	default:
		typ = reflect.TypeOf(collection)
	}
	SetSerializer(name, JavaCollectionSerializer{})
	RegisterPOJO(collection)
	collectionTypeMap[name] = typ
}

func getCollectionSerialize(name string) reflect.Type {
	return collectionTypeMap[name]
}

func isCollectionSerialize(name string) bool {
	return getCollectionSerialize(name) != nil
}

type JavaCollectionSerializer struct {
}

func (JavaCollectionSerializer) EncObject(e *Encoder, vv POJO) error {
	var (
		err error
	)
	v, ok := vv.(JavaCollectionObject)
	if !ok {
		return perrors.New("can not be converted into java collection object")
	}
	collectionName := v.JavaClassName()
	if collectionName == "" {
		return perrors.New("collection name empty")
	}
	list := v.Get()
	length := len(list)
	typeName := v.JavaClassName()
	err = writeCollectionBegin(length, typeName, e)
	if err != nil {
		return err
	}
	for i := 0; i < length; i++ {
		if err = e.Encode(list[i]); err != nil {
			return err
		}
	}
	return nil
}

func (JavaCollectionSerializer) DecObject(d *Decoder, typ reflect.Type, cls classInfo) (interface{}, error) {
	//for the java impl of hessian encode collections as list, which will not be decoded as object in go impl, this method should not be called
	return nil, perrors.New("unexpected collection decode call")
}

func (d *Decoder) decodeCollection(length int, listTyp string) (interface{}, error) {
	typ := getCollectionSerialize(listTyp)
	if typ == nil {
		return nil, perrors.New("no collection deserialize set as " + listTyp)
	}
	v := reflect.New(typ).Interface()
	collcetionV, ok := v.(JavaCollectionObject)
	if !ok {
		return nil, perrors.New("collection deserialize err " + listTyp)
	}
	list, err := d.readTypedListValue(length, "", false)
	if err != nil {
		return nil, err
	}
	listInterface, err := EnsureInterface(list, nil)
	if err != nil {
		return nil, err
	}
	listV, listOk := listInterface.([]interface{})
	if !listOk {
		return nil, perrors.New("collection deserialize err " + listTyp)
	}
	collcetionV.Set(listV)
	return collcetionV, nil
}

func writeCollectionBegin(length int, typeName string, e *Encoder) error {
	var err error
	if length <= int(LIST_DIRECT_MAX) {
		e.Append([]byte{BC_LIST_DIRECT + byte(length)})
		err = e.Encode(typeName)
		if err != nil {
			return err
		}
	} else {
		e.Append([]byte{BC_LIST_FIXED})
		err = e.Encode(typeName)
		if err != nil {
			return err
		}
		err = e.Encode(int32(length))
		if err != nil {
			return nil
		}
	}
	return nil
}
