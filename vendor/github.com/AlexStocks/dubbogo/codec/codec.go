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
