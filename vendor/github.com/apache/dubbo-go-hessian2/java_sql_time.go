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
	"io"
	"reflect"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go-hessian2/java_sql_time"
)

func init() {
	RegisterPOJO(&java_sql_time.Date{})
	RegisterPOJO(&java_sql_time.Time{})
	SetJavaSqlTimeSerialize(&java_sql_time.Date{})
	SetJavaSqlTimeSerialize(&java_sql_time.Time{})
}

// SetJavaSqlTimeSerialize register serializer for java.sql.Time & java.sql.Date
func SetJavaSqlTimeSerialize(time java_sql_time.JavaSqlTime) {
	name := time.JavaClassName()
	SetSerializer(name, JavaSqlTimeSerializer{})
}

// JavaSqlTimeSerializer used to encode & decode java.sql.Time & java.sql.Date
type JavaSqlTimeSerializer struct {
}

// nolint
func (JavaSqlTimeSerializer) EncObject(e *Encoder, vv POJO) error {

	var (
		i         int
		idx       int
		err       error
		clsDef    classInfo
		className string
		ptrV      reflect.Value
	)

	// ensure ptrV is pointer to know vv is type JavaSqlTime or not
	ptrV = reflect.ValueOf(vv)
	if reflect.TypeOf(vv).Kind() != reflect.Ptr {
		ptrV = PackPtr(ptrV)
	}
	v, ok := ptrV.Interface().(java_sql_time.JavaSqlTime)
	if !ok {
		return perrors.New("can not be converted into java sql time object")
	}
	className = v.JavaClassName()
	if className == "" {
		return perrors.New("class name empty")
	}
	tValue := reflect.ValueOf(vv)
	// check ref
	if n, ok := e.checkRefMap(tValue); ok {
		e.buffer = encRef(e.buffer, n)
		return nil
	}

	// write object definition
	idx = -1
	for i = range e.classInfoList {
		if v.JavaClassName() == e.classInfoList[i].javaName {
			idx = i
			break
		}
	}
	if idx == -1 {
		idx, ok = checkPOJORegistry(typeof(vv))
		if !ok {
			idx = RegisterPOJO(v)
		}
		_, clsDef, err = getStructDefByIndex(idx)
		if err != nil {
			return perrors.WithStack(err)
		}
		idx = len(e.classInfoList)
		e.classInfoList = append(e.classInfoList, clsDef)
		e.buffer = append(e.buffer, clsDef.buffer...)
	}
	e.buffer = e.buffer[0 : len(e.buffer)-1]
	e.buffer = encInt32(e.buffer, 1)
	e.buffer = encString(e.buffer, "value")

	// write object instance
	if byte(idx) <= OBJECT_DIRECT_MAX {
		e.buffer = encByte(e.buffer, byte(idx)+BC_OBJECT_DIRECT)
	} else {
		e.buffer = encByte(e.buffer, BC_OBJECT)
		e.buffer = encInt32(e.buffer, int32(idx))
	}
	e.buffer = encDateInMs(e.buffer, v.GetTime())
	return nil
}

// nolint
func (JavaSqlTimeSerializer) DecObject(d *Decoder, typ reflect.Type, cls classInfo) (interface{}, error) {

	if typ.Kind() != reflect.Struct {
		return nil, perrors.Errorf("wrong type expect Struct but get:%s", typ.String())
	}

	vRef := reflect.New(typ)
	// add pointer ref so that ref the same object
	d.appendRefs(vRef.Interface())

	tag, err := d.readByte()
	if err == io.EOF {
		return nil, err
	}
	date, err := d.decDate(int32(tag))
	if err != nil {
		return nil, perrors.WithStack(err)
	}
	sqlTime := vRef.Interface()

	result, ok := sqlTime.(java_sql_time.JavaSqlTime)
	if !ok {
		panic("result type is not sql time, please check the whether the conversion is ok")
	}
	result.SetTime(date)
	return result, nil
}
