#!/usr/bin/env bash
# ******************************************************
# DESC    :
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2017-09-04 22:53
# FILE    : pb.sh
# ******************************************************

mkdir ./src

# descriptor.proto
gopath=~/test/golang/lib/src/github.com/gogo/protobuf/protobuf
# If you are using any gogo.proto extensions you will need to specify the
# proto_path to include the descriptor.proto and gogo.proto.
# gogo.proto is located in github.com/gogo/protobuf/gogoproto
gogopath=~/test/golang/lib/src/

# protoc -I=$gopath:$gogopath:/Users/alex/test/golang/lib/src/github.com/AlexStocks/goext/database/redis/:./ --gogoslick_out=Mredis_meta.proto="github.com/AlexStocks/goext/database/redis":../app/  cluster_meta.proto
# protoc -I=$gopath:$gogopath:/Users/alex/test/golang/lib/src/github.com/AlexStocks/goext/database/redis/:./ --gogoslick_out=Mredis_meta.proto="github.com/AlexStocks/goext/database/redis":../app/  response.proto
# protoc -I=$gopath:$gogopath:./ --gogoslick_out=Mrole.proto="github.com/AlexStocks/goext/database/registry":./src/  service.proto
protoc -I=$gopath:$gogopath:./ --gogoslick_out=./src/  service.proto
