#!/bin/bash

cd ../../../../cmd/mosn/main
go build -gcflags "all=-N -l"

cd -

if [ ! -f ./mosn ]; then
  ln -s ../../../../cmd/mosn/main/main ./mosn
fi