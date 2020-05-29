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
	"strings"
	"sync"
)

import (
	perrors "github.com/pkg/errors"
)

// get @v go struct name
func typeof(v interface{}) string {
	return reflect.TypeOf(v).String()
}

/////////////////////////////////////////
// map/object
/////////////////////////////////////////

//class-def  ::= 'C' string int string* //  mandatory type string, the number of fields, and the field names.
//object     ::= 'O' int value* // class-def id, value list
//           ::= [x60-x6f] value* // class-def id, value list
//
//Object serialization
//
//class Car {
//  String color;
//  String model;
//}
//
//out.writeObject(new Car("red", "corvette"));
//out.writeObject(new Car("green", "civic"));
//
//---
//
//C                        # object definition (#0)
//  x0b example.Car        # type is example.Car
//  x92                    # two fields
//  x05 color              # color field name
//  x05 model              # model field name
//
//O                        # object def (long form)
//  x90                    # object definition #0
//  x03 red                # color field value
//  x08 corvette           # model field value
//
//x60                      # object def #0 (short form)
//  x05 green              # color field value
//  x05 civic              # model field value
//
//enum Color {
//  RED,
//  GREEN,
//  BLUE,
//}
//
//out.writeObject(Color.RED);
//out.writeObject(Color.GREEN);
//out.writeObject(Color.BLUE);
//out.writeObject(Color.GREEN);
//
//---
//
//C                         # class definition #0
//  x0b example.Color       # type is example.Color
//  x91                     # one field
//  x04 name                # enumeration field is "name"
//
//x60                       # object #0 (class def #0)
//  x03 RED                 # RED value
//
//x60                       # object #1 (class def #0)
//  x90                     # object definition ref #0
//  x05 GREEN               # GREEN value
//
//x60                       # object #2 (class def #0)
//  x04 BLUE                # BLUE value
//
//x51 x91                   # object ref #1, i.e. Color.GREEN
func (e *Encoder) encObject(v POJO) error {
	var (
		ok     bool
		i      int
		idx    int
		num    int
		err    error
		clsDef classInfo
	)

	vv := reflect.ValueOf(v)
	// check ref
	if n, ok := e.checkRefMap(vv); ok {
		e.buffer = encRef(e.buffer, n)
		return nil
	}

	vv = UnpackPtr(vv)
	// check nil pointer
	if !vv.IsValid() {
		e.buffer = encNull(e.buffer)
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
		idx, ok = checkPOJORegistry(typeof(v))
		if !ok {
			if reflect.TypeOf(v).Implements(javaEnumType) {
				idx = RegisterJavaEnum(v.(POJOEnum))
			} else {
				idx = RegisterPOJO(v)
			}
		}
		_, clsDef, err = getStructDefByIndex(idx)
		if err != nil {
			return perrors.WithStack(err)
		}

		idx = len(e.classInfoList)
		e.classInfoList = append(e.classInfoList, clsDef)
		e.buffer = append(e.buffer, clsDef.buffer...)
	}

	// write object instance
	if byte(idx) <= OBJECT_DIRECT_MAX {
		e.buffer = encByte(e.buffer, byte(idx)+BC_OBJECT_DIRECT)
	} else {
		e.buffer = encByte(e.buffer, BC_OBJECT)
		e.buffer = encInt32(e.buffer, int32(idx))
	}

	if reflect.TypeOf(v).Implements(javaEnumType) {
		e.buffer = encString(e.buffer, v.(POJOEnum).String())
		return nil
	}

	structs := []reflect.Value{vv}
	for len(structs) > 0 {
		vv := structs[0]
		vvt := vv.Type()
		num = vv.NumField()
		for i = 0; i < num; i++ {
			tf := vvt.Field(i)
			// skip unexported anonymous field
			if tf.PkgPath != "" {
				continue
			}

			// skip ignored field
			if tag, _ := tf.Tag.Lookup(tagIdentifier); tag == `-` {
				continue
			}

			field := vv.Field(i)
			if tf.Anonymous && field.Kind() == reflect.Struct {
				structs = append(structs, field)
				continue
			}

			if err = e.Encode(field.Interface()); err != nil {
				fieldName := field.Type().String()
				return perrors.Wrapf(err, "failed to encode field: %s, %+v", fieldName, field.Interface())
			}
		}

		structs = structs[1:]
	}

	return nil
}

/////////////////////////////////////////
// Object
/////////////////////////////////////////

//class-def  ::= 'C' string int string* //  mandatory type string, the number of fields, and the field names.
//object     ::= 'O' int value* // class-def id, value list
//           ::= [x60-x6f] value* // class-def id, value list
//
//Object serialization
//
//class Car {
//  String color;
//  String model;
//}
//
//out.writeObject(new Car("red", "corvette"));
//out.writeObject(new Car("green", "civic"));
//
//---
//
//C                        # object definition (#0)
//  x0b example.Car        # type is example.Car
//  x92                    # two fields
//  x05 color              # color field name
//  x05 model              # model field name
//
//O                        # object def (long form)
//  x90                    # object definition #0
//  x03 red                # color field value
//  x08 corvette           # model field value
//
//x60                      # object def #0 (short form)
//  x05 green              # color field value
//  x05 civic              # model field value
//
//
//
//
//
//enum Color {
//  RED,
//  GREEN,
//  BLUE,
//}
//
//out.writeObject(Color.RED);
//out.writeObject(Color.GREEN);
//out.writeObject(Color.BLUE);
//out.writeObject(Color.GREEN);
//
//---
//
//C                         # class definition #0
//  x0b example.Color       # type is example.Color
//  x91                     # one field
//  x04 name                # enumeration field is "name"
//
//x60                       # object #0 (class def #0)
//  x03 RED                 # RED value
//
//x60                       # object #1 (class def #0)
//  x90                     # object definition ref #0
//  x05 GREEN               # GREEN value
//
//x60                       # object #2 (class def #0)
//  x04 BLUE                # BLUE value
//
//x51 x91                   # object ref #1, i.e. Color.GREEN

func (d *Decoder) decClassDef() (interface{}, error) {
	var (
		err       error
		clsName   string
		fieldNum  int32
		fieldName string
		fieldList []string
	)

	clsName, err = d.decString(TAG_READ)
	if err != nil {
		return nil, perrors.WithStack(err)
	}
	fieldNum, err = d.decInt32(TAG_READ)
	if err != nil {
		return nil, perrors.WithStack(err)
	}
	fieldList = make([]string, fieldNum)
	for i := 0; i < int(fieldNum); i++ {
		fieldName, err = d.decString(TAG_READ)
		if err != nil {
			return nil, perrors.Wrapf(err, "decClassDef->decString, field num:%d, index:%d", fieldNum, i)
		}
		fieldList[i] = fieldName
	}

	return classInfo{javaName: clsName, fieldNameList: fieldList}, nil
}

type fieldInfo struct {
	indexes []int
	field   *reflect.StructField
}

// map[rType][fieldName]indexes
var fieldIndexCache sync.Map

func findFieldWithCache(name string, typ reflect.Type) ([]int, *reflect.StructField, error) {
	typCache, _ := fieldIndexCache.Load(typ)
	if typCache == nil {
		typCache = &sync.Map{}
		fieldIndexCache.Store(typ, typCache)
	}

	iindexes, existCache := typCache.(*sync.Map).Load(name)
	if existCache && iindexes != nil {
		finfo := iindexes.(*fieldInfo)
		var err error
		if len(finfo.indexes) == 0 {
			err = perrors.Errorf("failed to find field %s", name)
		}
		return finfo.indexes, finfo.field, err
	}

	indexes, field, err := findField(name, typ)
	typCache.(*sync.Map).Store(name, &fieldInfo{indexes: indexes, field: field})
	return indexes, field, err
}

// findField find structField in rType
//
// return
// 	indexes []int
// 	field reflect.StructField
// 	err error
func findField(name string, typ reflect.Type) ([]int, *reflect.StructField, error) {
	for i := 0; i < typ.NumField(); i++ {
		// matching tag first, then lowerCamelCase, SameCase, lowerCase

		typField := typ.Field(i)

		tagVal, hasTag := typField.Tag.Lookup(tagIdentifier)

		fieldName := typField.Name
		if hasTag && tagVal == name ||
			fieldName == name ||
			lowerCamelCase(fieldName) == name ||
			strings.ToLower(fieldName) == name {

			return []int{i}, &typField, nil
		}

		if typField.Anonymous && typField.Type.Kind() == reflect.Struct {
			next, field, _ := findField(name, typField.Type)
			if len(next) > 0 {
				indexes := []int{i}
				indexes = append(indexes, next...)

				return indexes, field, nil
			}
		}
	}

	return []int{}, nil, perrors.Errorf("failed to find field %s", name)
}

func (d *Decoder) decInstance(typ reflect.Type, cls classInfo) (interface{}, error) {
	if typ.Kind() != reflect.Struct {
		return nil, perrors.Errorf("wrong type expect Struct but get:%s", typ.String())
	}

	vRef := reflect.New(typ)
	// add pointer ref so that ref the same object
	d.appendRefs(vRef.Interface())

	vv := vRef.Elem()
	for i := 0; i < len(cls.fieldNameList); i++ {
		fieldName := cls.fieldNameList[i]

		index, fieldStruct, err := findFieldWithCache(fieldName, typ)
		if err != nil {
			return nil, perrors.Errorf("can not find field %s", fieldName)
		}

		// skip unexported anonymous field
		if fieldStruct.PkgPath != "" {
			continue
		}

		field := vv.FieldByIndex(index)
		if !field.CanSet() {
			return nil, perrors.Errorf("decInstance CanSet false for field %s", fieldName)
		}

		// get field type from type object, not do that from value
		fldTyp := UnpackPtrType(field.Type())

		// unpack pointer to enable value setting
		fldRawValue := UnpackPtrValue(field)

		kind := fldTyp.Kind()
		switch kind {
		case reflect.String:
			str, err := d.decString(TAG_READ)
			if err != nil {
				return nil, perrors.Wrapf(err, "decInstance->ReadString: %s", fieldName)
			}
			fldRawValue.SetString(str)

		case reflect.Int32, reflect.Int16, reflect.Int8:
			num, err := d.decInt32(TAG_READ)
			if err != nil {
				// java enum
				if fldRawValue.Type().Implements(javaEnumType) {
					d.unreadByte() // Enum parsing, decInt64 above has read a byte, so you need to return a byte here
					s, err := d.DecodeValue()
					if err != nil {
						return nil, perrors.Wrapf(err, "decInstance->decObject field name:%s", fieldName)
					}
					enumValue, _ := s.(JavaEnum)
					num = int32(enumValue)
				} else {
					return nil, perrors.Wrapf(err, "decInstance->decInt32, field name:%s", fieldName)
				}
			}
			fldRawValue.SetInt(int64(num))
		case reflect.Uint16, reflect.Uint8:
			num, err := d.decInt32(TAG_READ)
			if err != nil {
				return nil, perrors.Wrapf(err, "decInstance->decInt32, field name:%s", fieldName)
			}
			fldRawValue.SetUint(uint64(num))
		case reflect.Uint, reflect.Int, reflect.Int64:
			num, err := d.decInt64(TAG_READ)
			if err != nil {
				if fldTyp.Implements(javaEnumType) {
					d.unreadByte() // Enum parsing, decInt64 above has read a byte, so you need to return a byte here
					s, err := d.Decode()
					if err != nil {
						return nil, perrors.Wrapf(err, "decInstance->decObject field name:%s", fieldName)
					}
					enumValue, _ := s.(JavaEnum)
					num = int64(enumValue)
				} else {
					return nil, perrors.Wrapf(err, "decInstance->decInt64 field name:%s", fieldName)
				}
			}

			fldRawValue.SetInt(num)
		case reflect.Uint32, reflect.Uint64:
			num, err := d.decInt64(TAG_READ)
			if err != nil {
				return nil, perrors.Wrapf(err, "decInstance->decInt64, field name:%s", fieldName)
			}
			fldRawValue.SetUint(uint64(num))
		case reflect.Bool:
			b, err := d.Decode()
			if err != nil {
				return nil, perrors.Wrapf(err, "decInstance->Decode field name:%s", fieldName)
			}
			v, ok := b.(bool)
			if !ok {
				return nil, perrors.Wrapf(err, "value convert to bool failed, field name:%s", fieldName)
			}

			if fldRawValue.Kind() == reflect.Ptr && fldRawValue.CanSet() {
				if b != nil {
					field.Set(reflect.ValueOf(&v))
				}
			} else if fldRawValue.Kind() != reflect.Ptr {
				fldRawValue.SetBool(v)
			}

		case reflect.Float32, reflect.Float64:
			num, err := d.decDouble(TAG_READ)
			if err != nil {
				return nil, perrors.Wrapf(err, "decInstance->decDouble field name:%s", fieldName)
			}
			fldRawValue.SetFloat(num.(float64))

		case reflect.Map:
			// decode map should use the original field value for correct value setting
			err := d.decMapByValue(field)
			if err != nil {
				return nil, perrors.Wrapf(err, "decInstance->decMapByValue field name: %s", fieldName)
			}

		case reflect.Slice, reflect.Array:
			m, err := d.decList(TAG_READ)
			if err != nil {
				if err == io.EOF {
					break
				}
				return nil, perrors.WithStack(err)
			}

			// set slice separately
			err = SetSlice(fldRawValue, m)
			if err != nil {
				return nil, err
			}
		case reflect.Struct, reflect.Interface:
			var (
				err error
				s   interface{}
			)
			typ := UnpackPtrType(fldRawValue.Type())
			if typ.String() == "time.Time" {
				s, err = d.decDate(TAG_READ)
				if err != nil {
					return nil, perrors.WithStack(err)
				}
				SetValue(fldRawValue, EnsurePackValue(s))
			} else {
				s, err = d.decObject(TAG_READ)
				if err != nil {
					return nil, perrors.WithStack(err)
				}
				if s != nil {
					// set value which accepting pointers
					SetValue(fldRawValue, EnsurePackValue(s))
				}
			}

		default:
			return nil, perrors.Errorf("unknown struct member type: %v %v", kind, typ.Name()+"."+fieldStruct.Name)
		}
	} // end for

	return vRef.Interface(), nil
}

func (d *Decoder) appendClsDef(cd classInfo) {
	d.classInfoList = append(d.classInfoList, cd)
}

func (d *Decoder) getStructDefByIndex(idx int) (reflect.Type, classInfo, error) {
	var (
		ok  bool
		cls classInfo
		s   structInfo
		err error
	)

	if len(d.classInfoList) <= idx || idx < 0 {
		return nil, cls, perrors.Errorf("illegal class index @idx %d", idx)
	}
	cls = d.classInfoList[idx]
	s, ok = getStructInfo(cls.javaName)
	if !ok {
		if !d.isSkip {
			err = perrors.Errorf("can not find go type name %s in registry", cls.javaName)
		}
		return nil, cls, err
	}

	return s.typ, cls, nil
}

func (d *Decoder) decEnum(javaName string, flag int32) (JavaEnum, error) {
	var (
		err       error
		enumName  string
		ok        bool
		info      structInfo
		enumValue JavaEnum
	)
	enumName, err = d.decString(TAG_READ) // java enum class member is "name"
	if err != nil {
		return InvalidJavaEnum, perrors.Wrap(err, "decString for decJavaEnum")
	}
	info, ok = getStructInfo(javaName)
	if !ok {
		return InvalidJavaEnum, perrors.Errorf("getStructInfo(javaName:%s) = false", javaName)
	}

	enumValue = info.inst.(POJOEnum).EnumValue(enumName)
	d.appendRefs(enumValue)
	return enumValue, nil
}

// skip this object
func (d *Decoder) skip(cls classInfo) error {
	len := len(cls.fieldNameList)
	if len < 1 {
		return nil
	}

	for i := 0; i < len; i++ {
		// skip class fields.
		if _, err := d.DecodeValue(); err != nil {
			return err
		}
	}

	return nil
}

func (d *Decoder) decObject(flag int32) (interface{}, error) {
	var (
		tag byte
		idx int32
		err error
		typ reflect.Type
		cls classInfo
	)

	if flag != TAG_READ {
		tag = byte(flag)
	} else {
		tag, _ = d.readByte()
	}

	switch {
	case tag == BC_NULL:
		return nil, nil
	case tag == BC_REF:
		return d.decRef(int32(tag))
	case tag == BC_OBJECT_DEF:
		clsDef, err := d.decClassDef()
		if err != nil {
			return nil, perrors.Wrap(err, "decObject->decClassDef byte double")
		}
		cls, _ = clsDef.(classInfo)
		//add to slice
		d.appendClsDef(cls)

		return d.DecodeValue()

	case tag == BC_OBJECT:
		idx, err = d.decInt32(TAG_READ)
		if err != nil {
			return nil, err
		}

		typ, cls, err = d.getStructDefByIndex(int(idx))
		if err != nil {
			return nil, err
		}
		if typ == nil {
			return nil, d.skip(cls)
		}
		if typ.Implements(javaEnumType) {
			return d.decEnum(cls.javaName, TAG_READ)
		}

		if c, ok := GetSerializer(cls.javaName); ok {
			return c.DecObject(d, typ, cls)
		}

		return d.decInstance(typ, cls)

	case BC_OBJECT_DIRECT <= tag && tag <= (BC_OBJECT_DIRECT+OBJECT_DIRECT_MAX):
		typ, cls, err = d.getStructDefByIndex(int(tag - BC_OBJECT_DIRECT))
		if err != nil {
			return nil, err
		}
		if typ == nil {
			return nil, d.skip(cls)
		}
		if typ.Implements(javaEnumType) {
			return d.decEnum(cls.javaName, TAG_READ)
		}

		if c, ok := GetSerializer(cls.javaName); ok {
			return c.DecObject(d, typ, cls)
		}

		return d.decInstance(typ, cls)

	default:
		return nil, perrors.Errorf("decObject illegal object type tag:%+v", tag)
	}
}
