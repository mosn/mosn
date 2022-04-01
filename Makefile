SHELL = /bin/bash

TARGET          = mosnd
TARGET_SIDECAR  = mosn
CONFIG_FILE     = mosn_config.json
PROJECT_NAME    = mosn.io/mosn

# default istio version
ISTIO_VERSION   = $(shell cat ISTIO_VERSION)

SCRIPT_DIR      = $(shell pwd)/etc/script

MAJOR_VERSION   = $(shell cat VERSION)
GIT_VERSION     = $(shell git log -1 --pretty=format:%h)
GIT_NOTES       = $(shell git log -1 --oneline)

BUILD_IMAGE     = golang:1.14.13

WASM_IMAGE      = mosn-wasm

IMAGE_NAME      = mosn
REPOSITORY      = mosnio/${IMAGE_NAME}

RPM_BUILD_IMAGE = afenp-rpm-builder
RPM_VERSION     = $(shell cat VERSION | tr -d '-')
RPM_TAR_NAME    = afe-${TARGET}
RPM_SRC_DIR     = ${RPM_TAR_NAME}-${RPM_VERSION}
RPM_TAR_FILE    = ${RPM_SRC_DIR}.tar.gz

TAGS			= ${tags}
TAGS_OPT 		=

# support build custom tags
ifneq ($(TAGS),)
TAGS_OPT 		= -tags ${TAGS}
endif

ut-local:
	GO111MODULE=on go test -gcflags=-l -v `go list ./pkg/... | grep -v pkg/mtls/crypto/tls | grep -v pkg/networkextention`
	make unit-test-istio-${ISTIO_VERSION}

unit-test:
	docker run --rm -v $(shell go env GOPATH):/go -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make ut-local

coverage-local:
	sh ${SCRIPT_DIR}/report.sh

coverage:
	docker run --rm -v $(shell go env GOPATH):/go -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make coverage-local

integrate-local:
	GO111MODULE=on go test -p 1 -v ./test/integrate/...

integrate-local-netpoll:
	GO111MODULE=on NETPOLL=on go test -p 1 -v ./test/integrate/...

integrate:
	docker run --rm -v $(shell go env GOPATH):/go -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make integrate-local


integrate-netpoll:
	docker run --rm -v $(shell go env GOPATH):/go -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make integrate-local-netpoll

integrate-framework:
	@cd ./test/cases && bash run_all.sh

integrate-new:
	docker run --rm -v $(shell go env GOPATH):/go -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make integrate-framework

build:
	docker run --rm -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make build-local

build-wasm-image:
	docker build --rm -t ${WASM_IMAGE}:${MAJOR_VERSION} -f build/contrib/builder/wasm/Dockerfile .

binary: build

build-local:
	@rm -rf build/bundles/${MAJOR_VERSION}/binary
	GO111MODULE=on CGO_ENABLED=1 go build ${TAGS_OPT} \
		-ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') -X main.Version=${MAJOR_VERSION}(${GIT_VERSION}) -X ${PROJECT_NAME}/pkg/types.IstioVersion=${ISTIO_VERSION}" \
		-v -o ${TARGET} \
		${PROJECT_NAME}/cmd/mosn/main
	mkdir -p build/bundles/${MAJOR_VERSION}/binary
	mv ${TARGET} build/bundles/${MAJOR_VERSION}/binary
	@cd build/bundles/${MAJOR_VERSION}/binary && $(shell which md5sum) -b ${TARGET} | cut -d' ' -f1  > ${TARGET}.md5
	cp configs/${CONFIG_FILE} build/bundles/${MAJOR_VERSION}/binary
	cp build/bundles/${MAJOR_VERSION}/binary/${TARGET}  build/bundles/${MAJOR_VERSION}/binary/${TARGET_SIDECAR}

image:
	@rm -rf IMAGEBUILD
	cp -r build/contrib/builder/image IMAGEBUILD && cp build/bundles/${MAJOR_VERSION}/binary/${TARGET} IMAGEBUILD && cp -r configs IMAGEBUILD && cp -r etc IMAGEBUILD
	docker build --no-cache --rm -t ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} IMAGEBUILD
	docker tag ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} ${REPOSITORY}:${MAJOR_VERSION}-${GIT_VERSION}
	rm -rf IMAGEBUILD

# change istio version support
istio-1.5.2:
	@echo 1.5.2 > ISTIO_VERSION
	@bash istio_ctrl.sh istio152
	@cp istio/istio152/main/* ./cmd/mosn/main/
	@go mod edit -replace github.com/envoyproxy/go-control-plane=github.com/envoyproxy/go-control-plane@v0.9.4
	@go mod edit -replace istio.io/api=istio.io/api@v0.0.0-20200227213531-891bf31f3c32
	@go mod tidy

istio-1.10.6:
	@echo 1.10.6 > ISTIO_VERSION
	@bash istio_ctrl.sh istio1100
	@cp istio/istio1100/main/* ./cmd/mosn/main/
	@go mod edit -replace istio.io/api=istio.io/api@v0.0.0-20211103171850-665ed2b92d52
	@go mod edit -replace github.com/envoyproxy/go-control-plane=github.com/envoyproxy/go-control-plane@v0.10.0
	@go mod tidy

# istio test
unit-test-istio:
	GO111MODULE=on go test -gcflags="all=-N -l" -v `go list ./istio/...`

	

.PHONY: unit-test build image rpm upload shell
