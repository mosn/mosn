// Copyright (c) 2015 Asim Aslam.
// Copyright (c) 2016 ~ 2018, Alex Stocks.
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

package common

import (
	"encoding/json"
	"net/http"
)

// Errors provide a way to return detailed information
// for an RPC request error. The error is normally
// JSON encoded.
type Error struct {
	ID     string `json:"id"`
	Code   int32  `json:"code"`
	Detail string `json:"detail"`
	Status string `json:"status"`
}

func (e *Error) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func NewError(id, detail string, code int32) error {
	return &Error{
		ID:     id,
		Code:   code,
		Detail: detail,
		Status: http.StatusText(int(code)),
	}
}

func Parse(err string) *Error {
	e := new(Error)
	errr := json.Unmarshal([]byte(err), e)
	if errr != nil {
		e.Detail = err
	}
	return e
}

func BadRequest(id, detail string) error {
	return &Error{
		ID:     id,
		Code:   400,
		Detail: detail,
		Status: http.StatusText(400),
	}
}

func Unauthorized(id, detail string) error {
	return &Error{
		ID:     id,
		Code:   401,
		Detail: detail,
		Status: http.StatusText(401),
	}
}

func Forbidden(id, detail string) error {
	return &Error{
		ID:     id,
		Code:   403,
		Detail: detail,
		Status: http.StatusText(403),
	}
}

func NotFound(id, detail string) error {
	return &Error{
		ID:     id,
		Code:   404,
		Detail: detail,
		Status: http.StatusText(404),
	}
}

func InternalServerError(id, detail string) error {
	return &Error{
		ID:     id,
		Code:   500,
		Detail: detail,
		Status: http.StatusText(500),
	}
}
