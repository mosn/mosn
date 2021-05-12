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

package abi

import (
	"sync"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// Factory is the ABI factory func.
type Factory func(instance types.WasmInstance) types.ABI

// string -> Factory.
var abiMap = sync.Map{}

// RegisterABI registers an abi factory.
func RegisterABI(name string, factory Factory) {
	abiMap.Store(name, factory)
}

func GetABI(instance types.WasmInstance, name string) types.ABI {
	if instance == nil || name == "" {
		log.DefaultLogger.Errorf("[abi][registry] GetABI invalid param, name: %v, instance: %v", name, instance)
		return nil
	}

	v, ok := abiMap.Load(name)
	if !ok {
		log.DefaultLogger.Errorf("[abi][registry] GetABI not found in registry, name: %v", name)
		return nil
	}

	abiNameList := instance.GetModule().GetABINameList()
	for _, abi := range abiNameList {
		if name == abi {
			factory := v.(Factory)
			return factory(instance)
		}
	}

	log.DefaultLogger.Errorf("[abi][register] GetABI not found in wasm instance, name: %v", name)

	return nil
}

func GetABIList(instance types.WasmInstance) []types.ABI {
	if instance == nil {
		log.DefaultLogger.Errorf("[abi][registry] GetABIList nil instance: %v", instance)
		return nil
	}

	res := make([]types.ABI, 0)

	abiNameList := instance.GetModule().GetABINameList()
	for _, abiName := range abiNameList {
		v, ok := abiMap.Load(abiName)
		if !ok {
			log.DefaultLogger.Warnf("[abi][registry] GetABIList abi not registered, name: %v", abiName)
			continue
		}

		factory := v.(Factory)
		res = append(res, factory(instance))
	}

	return res
}
