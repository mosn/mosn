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

package ext

type ServerLister interface {
	List(string) chan []string
}

var serverListers = map[string]ServerLister{}

func RegisterServerLister(name string, lister ServerLister) {
	serverListers[name] = lister
}

func GetServerLister(name string) ServerLister {
	return serverListers[name]
}

// ConnectionValidator
type ConnectionValidator interface {
	Validate(credential string, host string, cluster string) bool
}

func RegisterConnectionValidator(name string, validator ConnectionValidator) {
	connectionValidators[name] = validator
}

func GetConnectionValidator(name string) ConnectionValidator {
	return connectionValidators[name]
}

var connectionValidators = map[string]ConnectionValidator{}

// ConnectionCredentialGetter
type ConnectionCredentialGetter func(cluster string) string

func RegisterConnectionCredentialGetter(name string, getter ConnectionCredentialGetter) {
	connectionCredentialGetters[name] = getter
}

func GetConnectionCredentialGetter(name string) ConnectionCredentialGetter {
	return connectionCredentialGetters[name]
}

var connectionCredentialGetters = map[string]ConnectionCredentialGetter{}
