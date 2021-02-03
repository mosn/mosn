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

import "strconv"

// IllegalFormatPrecisionException represents an exception of the same name in java
type IllegalFormatPrecisionException struct {
	SerialVersionUID     int64
	DetailMessage        string
	SuppressedExceptions []Throwabler
	StackTrace           []StackTraceElement
	Cause                Throwabler
	P                    int32
}

// NewIllegalFormatPrecisionException is the constructor
func NewIllegalFormatPrecisionException(p int32) *IllegalFormatPrecisionException {
	return &IllegalFormatPrecisionException{P: p, StackTrace: []StackTraceElement{}}
}

// Error output error message
func (e IllegalFormatPrecisionException) Error() string {
	return strconv.Itoa(int(e.P))
}

// JavaClassName  java fully qualified path
func (IllegalFormatPrecisionException) JavaClassName() string {
	return "java.util.IllegalFormatPrecisionException"
}
