SHELL = /bin/bash

IMAGE_NAME      = mosn
MOSN_VERSION    = $(shell cat ../../VERSION)
#ISTIO_VERSION   = $(shell cat ../../ISTIO_VERSION)
IMAGE_NAME_ENVOY_SDK = mosn-network-envoy-sdk
IMAGE_NAME_MOSN_NETWORK_ENVOY = mosn-network-on-envoy
MOSNIOHUB = mosnio
ENVOY_VERSION = 1.16.2

TARGET          = output
PROJECT_NAME = mosn.io/mosn
PROJECT_NAME_NETWORKDIR = ${PROJECT_NAME}/pkg/networkextention

# build golang extention sdk
build-l7:
	@rm -rf ${TARGET}
	mkdir -p ${TARGET}
	GODEBUG=cgocheck=1 go build -tags mosn_debug -buildmode=c-shared \
		-v -o ${TARGET}/golang_extention.so ./mosn.go init.go l7_golang_extention.go

# build mosn on envoy image
build-moe-image:
	docker run --rm -v $(shell cd ../../ && pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME}  ${MOSNIOHUB}/${IMAGE_NAME_ENVOY_SDK}:${ENVOY_VERSION}  sh -c "cd /go/src/${PROJECT_NAME_NETWORKDIR}/ &&  make build-l7"
	@rm -rf IMAGEBUILD
	cp -r build/image/envoy IMAGEBUILD && cp ${TARGET}/golang_extention.so IMAGEBUILD/  
	docker build --no-cache --rm -t ${IMAGE_NAME}:${MOSN_VERSION} IMAGEBUILD
	docker tag ${IMAGE_NAME}:${MOSN_VERSION}  ${MOSNIOHUB}/${IMAGE_NAME_MOSN_NETWORK_ENVOY}:${MOSN_VERSION}
	@rm -rf IMAGEBUILD
	@rm -rf ${TARGET}

# build base image
build-moe-base-image:
	@rm -rf IMAGEBUILD
	mkdir -p IMAGEBUILD
	cp -r build/image/envoy/envoy IMAGEBUILD
	cp -r build/image/envoy/DockerBasefile IMAGEBUILD//Dockerfile
	docker build --no-cache --rm -t ${IMAGE_NAME_ENVOY_SDK}:${ENVOY_VERSION} IMAGEBUILD
	docker tag ${IMAGE_NAME_ENVOY_SDK}:${ENVOY_VERSION} ${MOSNIOHUB}/${IMAGE_NAME_ENVOY_SDK}:${ENVOY_VERSION}
	@rm -rf IMAGEBUILD
	@rm -rf ${TARGET}

clean:
	@rm -rf ${TARGET}
	@rm -rf IMAGEBUILD


# TODO upgrade mosn
#istio-1.5.2:
#	cp ./istio152/istio.go ./
