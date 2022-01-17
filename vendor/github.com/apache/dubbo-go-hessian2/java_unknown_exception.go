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
	"sync"
)

import (
	"github.com/apache/dubbo-go-hessian2/java_exception"
)

var exceptionCheckMutex sync.Mutex

func checkAndGetException(cls *classInfo) (*structInfo, bool) {
	if len(cls.fieldNameList) < 4 {
		return nil, false
	}
	var (
		throwable *structInfo
		ok        bool
	)
	count := 0
	for _, item := range cls.fieldNameList {
		if item == "detailMessage" || item == "suppressedExceptions" || item == "stackTrace" || item == "cause" {
			count++
		}
	}
	// if have these 4 fields, it is throwable struct
	if count == 4 {
		exceptionCheckMutex.Lock()
		defer exceptionCheckMutex.Unlock()
		if throwable, ok = getStructInfo(cls.javaName); ok {
			return throwable, true
		}
		RegisterPOJO(newBizException(cls.javaName))
		if throwable, ok = getStructInfo(cls.javaName); ok {
			return throwable, true
		}
	}
	return throwable, count == 4
}

type UnknownException struct {
	SerialVersionUID     int64
	DetailMessage        string
	SuppressedExceptions []java_exception.Throwabler
	StackTrace           []java_exception.StackTraceElement
	Cause                java_exception.Throwabler
	name                 string
}

// NewThrowable is the constructor
func newBizException(name string) *UnknownException {
	return &UnknownException{name: name, StackTrace: []java_exception.StackTraceElement{}}
}

// Error output error message
func (e UnknownException) Error() string {
	return fmt.Sprintf("throw %v : %v", e.name, e.DetailMessage)
}

// JavaClassName  java fully qualified path
func (e UnknownException) JavaClassName() string {
	return e.name
}

// equals to getStackTrace in java
func (e UnknownException) GetStackTrace() []java_exception.StackTraceElement {
	return e.StackTrace
}
