#!/bin/bash

set -x

function make_build {
	mkdir build
	cp ../../../cmd/moe/main/* build
	cp filter.go build
	cd build
	GO111MODULE=on CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build --buildmode=c-shared -v -o libmosn.so
	mv libmosn.so ../
	cd ../
	rm -rf build
}

make_build
