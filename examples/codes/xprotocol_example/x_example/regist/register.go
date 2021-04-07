package main

import (
	"fmt"
	"mosn.io/api"
	"x_example"
)

type Register struct {
	exampleStatusMapping x_example.StatusMapping

	exampleMatcher x_example.Matcher

	proto x_example.Proto
}

func (r Register) ProtocolName() api.ProtocolName {
	return r.proto.Name()
}

func (r Register) XProtocol() api.XProtocol {
	return &r.proto
}

func (r Register) ProtocolMatch() api.ProtocolMatch {
	return r.exampleMatcher.ExampleMatcher
}

func (r Register) HTTPMapping() api.HTTPMapping {
	return &r.exampleStatusMapping
}

var ExampleRegister = Register{}

//loader_func_name that go-Plugin use,LoadCodec is default name
func LoadCodec() api.XProtocolCodec {
	fmt.Println("插件注册成功")
	return ExampleRegister
}
