SHELL = /bin/bash

IMAGE_NAME      = mosn
MAJOR_VERSION   = $(shell cat ../../VERSION)

TARGET          = output
PROJECT_NAME = mosn.io/mosn/pkg/networkextention

# build golang extention sdk
build-l7:
	@rm -rf ${TARGET}
	mkdir -p ${TARGET}
	GODEBUG=cgocheck=1 go build -mod=vendor  -buildmode=c-shared \
		-v -o ${TARGET}/golang_extention.so ./mosn.go  init.go l7_golang_extention.go

# build mosn on envoy image
build-moe-image:
	@rm -rf ${TARGET}
	mkdir -p ${TARGET}
	GODEBUG=cgocheck=1 go build -mod=vendor  -buildmode=c-shared \
		-v -o ${TARGET}/golang_extention.so ./mosn.go  init.go l7_golang_extention.go

	cp -r build/image/envoy IMAGEBUILD && cp ${TARGET}/golang_extention.so IMAGEBUILD/  
	docker build --no-cache --rm -t ${IMAGE_NAME}:${MAJOR_VERSION} IMAGEBUILD
	@rm -rf IMAGEBUILD

