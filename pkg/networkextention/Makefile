SHELL = /bin/bash

IMAGE_NAME      = mosn
MOSN_VERSION   = $(shell cat ../../VERSION)
IMAGE_NAME_ENVOY_SDK = mosn-network-envoy-sdk
ENVOY_VERSION = 1.16.2

TARGET          = output
PROJECT_NAME = mosn.io/mosn/pkg/networkextention

# build golang extention sdk
build-l7:
	@rm -rf ${TARGET}
	mkdir -p ${TARGET}
	GODEBUG=cgocheck=1 go build -tags mosn_debug -buildmode=c-shared \
		-v -o ${TARGET}/golang_extention.so ./mosn.go  init.go l7_golang_extention.go

# build mosn on envoy image
build-moe-image:
	@rm -rf ${TARGET}
	mkdir -p ${TARGET}
	GODEBUG=cgocheck=1 go build  -tags mosn_debug -buildmode=c-shared \
		-v -o ${TARGET}/golang_extention.so ./mosn.go  init.go l7_golang_extention.go

	@rm -rf IMAGEBUILD
	cp -r build/image/envoy IMAGEBUILD && cp ${TARGET}/golang_extention.so IMAGEBUILD/  
	docker build --no-cache --rm -t ${IMAGE_NAME}:${MOSN_VERSION} IMAGEBUILD
	@rm -rf IMAGEBUILD
	@rm -rf ${TARGET}

# build base image
build-moe-base-image:
	@rm -rf IMAGEBUILD
	mkdir -p IMAGEBUILD
	cp -r build/image/envoy/envoy IMAGEBUILD
	cp -r build/image/envoy/DockerBasefile IMAGEBUILD//Dockerfile
	docker build --no-cache --rm -t ${IMAGE_NAME_ENVOY_SDK}:${ENVOY_VERSION} IMAGEBUILD
	@rm -rf IMAGEBUILD
	@rm -rf ${TARGET}

clean:
	@rm -rf ${TARGET}
	@rm -rf IMAGEBUILD
