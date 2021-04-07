#!/bin/bash

go build --buildmode=plugin  -o codec.so  \
  ./codec/api.go  \
  ./codec/command.go  \
  ./codec/decoder.go  \
  ./codec/encoder.go  \
  ./codec/mapping.go  \
  ./codec/matcher.go  \
  ./codec/protocol.go \
  ./codec/types.go