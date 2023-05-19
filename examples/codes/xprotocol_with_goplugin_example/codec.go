package main

import (
	"context"

	"mosn.io/api"
	"mosn.io/mosn/examples/codes/xprotocol_with_goplugin_example/codec"
)

type Codec struct {
	exampleStatusMapping codec.StatusMapping

	exampleMatcher codec.Matcher

	proto codec.Proto
}

func (r Codec) ProtocolName() api.ProtocolName {
	return r.proto.Name()
}

func (r Codec) NewXProtocol(context.Context) api.XProtocol {
	return &r.proto
}

func (r Codec) ProtocolMatch() api.ProtocolMatch {
	return r.exampleMatcher.ExampleMatcher
}

func (r Codec) HTTPMapping() api.HTTPMapping {
	return &r.exampleStatusMapping
}

// loader_func_name that go-Plugin use,LoadCodec is default name
func LoadCodec() api.XProtocolCodec {
	return &Codec{}
}
