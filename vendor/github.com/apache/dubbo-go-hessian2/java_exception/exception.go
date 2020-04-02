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

package java_exception

////////////////////////////
// Throwable interface
////////////////////////////

// Throwabler represents an exception of the same name in java
type Throwabler interface {
	Error() string
	JavaClassName() string
}

////////////////////////////
// Throwable
////////////////////////////
// Throwable represents an exception of the same name in java
type Throwable struct {
	SerialVersionUID     int64
	DetailMessage        string
	SuppressedExceptions []Throwabler
	StackTrace           []StackTraceElement
	Cause                Throwabler
}

// NewThrowable is the constructor
func NewThrowable(detailMessage string) *Throwable {
	return &Throwable{DetailMessage: detailMessage, StackTrace: []StackTraceElement{}}
}

// Error output error message
func (e Throwable) Error() string {
	return e.DetailMessage
}

//JavaClassName  java fully qualified path
func (Throwable) JavaClassName() string {
	return "java.lang.Throwable"
}

////////////////////////////
// Exception
////////////////////////////

type Exception struct {
	SerialVersionUID     int64
	DetailMessage        string
	SuppressedExceptions []Throwabler
	StackTrace           []StackTraceElement
	Cause                Throwabler
}

func NewException(detailMessage string) *Exception {
	return &Exception{DetailMessage: detailMessage, StackTrace: []StackTraceElement{}}
}

func (e Exception) Error() string {
	return e.DetailMessage
}

//JavaClassName  java fully qualified path
func (Exception) JavaClassName() string {
	return "java.lang.Exception"
}

////////////////////////////
// StackTraceElement
////////////////////////////

type StackTraceElement struct {
	DeclaringClass string
	MethodName     string
	FileName       string
	LineNumber     int
}

//JavaClassName  java fully qualified path
func (StackTraceElement) JavaClassName() string {
	return "java.lang.StackTraceElement"
}

type Class struct {
	Name string
}

type Method struct {
	Name string
}

//JavaClassName  java fully qualified path
func (Method) JavaClassName() string {
	return "java.lang.reflect.Method"
}

func (Class) JavaClassName() string {
	return "java.lang.Class"
}
