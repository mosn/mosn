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
	big "github.com/dubbogo/gost/math/big"
)

type bigInteger = big.Integer

func init() {
	RegisterPOJO(&bigInteger{})
	SetSerializer("java.math.BigInteger", IntegerSerializer{})

	RegisterPOJO(&big.Decimal{})
	SetSerializer("java.math.BigDecimal", DecimalSerializer{})
}

type Serializer interface {
	EncObject(*Encoder, POJO) error
	DecObject(*Decoder, reflect.Type, classInfo) (interface{}, error)
}

var serializerMap = make(map[string]Serializer, 16)

func SetSerializer(key string, codec Serializer) {
	serializerMap[key] = codec
}

func GetSerializer(key string) (Serializer, bool) {
	codec, ok := serializerMap[key]
	return codec, ok
}

type IntegerSerializer struct{}

func (IntegerSerializer) DecObject(d *Decoder, typ reflect.Type, cls classInfo) (interface{}, error) {
	bigInt, err := d.decInstance(typ, cls)
	if err != nil {
		return nil, err
	}

	result, ok := bigInt.(*bigInteger)
	if !ok {
		panic("result type is not Integer, please check the whether the conversion is ok")
	}

	result.FromSignAndMag(result.Signum, result.Mag)
	return result, nil
}

func (IntegerSerializer) EncObject(e *Encoder, v POJO) error {
	bigInt, ok := v.(bigInteger)
	if !ok {
		return e.encObject(v)
	}
	bigInt.Signum, bigInt.Mag = bigInt.GetSignAndMag()
	return e.encObject(bigInt)
}

type DecimalSerializer struct{}

func (DecimalSerializer) EncObject(e *Encoder, v POJO) error {
	decimal, ok := v.(big.Decimal)
	if !ok {
		return e.encObject(v)
	}
	decimal.Value = decimal.String()
	return e.encObject(decimal)
}

func (DecimalSerializer) DecObject(d *Decoder, typ reflect.Type, cls classInfo) (interface{}, error) {
	dec, err := d.decInstance(typ, cls)
	if err != nil {
		return nil, err
	}
	result, ok := dec.(*big.Decimal)
	if !ok {
		panic("result type is not decimal,please check the whether the conversion is ok")
	}
	err = result.FromString(result.Value)
	if err != nil {
		return nil, err
	}
	return result, nil
}
