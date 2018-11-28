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

package codec

import (
	"errors"
	"io"
	"time"
)

const (
	Error     MessageType = 0x01
	Request               = 0x02
	Response              = 0x04
	Heartbeat             = 0x08
)

var (
	ErrHeaderNotEnough = errors.New("header buffer too short")
	ErrBodyNotEnough   = errors.New("body buffer too short")
	ErrJavaException   = errors.New("got java exception")
	ErrIllegalPackage  = errors.New("illegal package!")
)

type MessageType int

// Takes in a connection/buffer and returns a new Codec
type NewCodec func(io.ReadWriteCloser) Codec

type Codec interface {
	ReadHeader(*Message, MessageType) error
	ReadBody(interface{}) error
	Write(m *Message, args interface{}) error
	Close() error
	String() string
}

type Message struct {
	ID          int64
	Version     string
	Type        MessageType
	ServicePath string // service path
	Target      string // Service
	Method      string
	Timeout     time.Duration // request timeout
	Error       string
	Header      map[string]string
	BodyLen     int
}
