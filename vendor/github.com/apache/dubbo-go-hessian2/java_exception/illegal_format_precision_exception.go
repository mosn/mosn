// Copyright 2016-2019 Yincheng Fang
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package java_exception

import "strconv"

type IllegalFormatPrecisionException struct {
	SerialVersionUID     int64
	DetailMessage        string
	SuppressedExceptions []Throwabler
	StackTrace           []StackTraceElement
	Cause                Throwabler
	P                    int32
}

func NewIllegalFormatPrecisionException(p int32) *IllegalFormatPrecisionException {
	return &IllegalFormatPrecisionException{P: p, StackTrace: []StackTraceElement{}}
}

func (e IllegalFormatPrecisionException) Error() string {
	return strconv.Itoa(int(e.P))
}

func (IllegalFormatPrecisionException) JavaClassName() string {
	return "java.util.IllegalFormatPrecisionException"
}
