SHELL = /bin/bash

TARGET       = mosnd
GIT_USER     = afe
PROJECT_NAME = gitlab.alipay-inc.com/${GIT_USER}/mosn

MAJOR_VERSION = $(shell cat VERSION)
GIT_VERSION   = $(shell git log -1 --pretty=format:%h)
GIT_NOTES     = $(shell git log -1 --oneline)

BUILD_IMAGE   = godep-builder

IMAGE_NAME = ${GIT_USER}/mosnd
REGISTRY = acs-reg.alipay.com

ut-local:
	go test ./...

unit-test:
	docker build --rm -t ${BUILD_IMAGE} contrib/builder/binary
	docker run --rm -v $(GOPATH):/go -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make ut-local

build:
	docker build --rm -t ${BUILD_IMAGE} contrib/builder/binary
	docker run --rm -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make build-local

binary: build

build-local:
	@rm -rf bundles/${MAJOR_VERSION}/binary
	CGO_ENABLED=0 go build \
		-ldflags "-X main.Version=${MAJOR_VERSION}(${GIT_VERSION})" \
		-v -o ${TARGET} \
		${PROJECT_NAME}/pkg/mosn
	mkdir -p bundles/${MAJOR_VERSION}/binary
	mv ${TARGET} bundles/${MAJOR_VERSION}/binary
	@cd bundles/${MAJOR_VERSION}/binary && $(shell which md5sum) -b ${TARGET} | cut -d' ' -f1  > ${TARGET}.md5

image:
	@rm -rf IMAGEBUILD
	cp -r contrib/builder/image IMAGEBUILD && cp bundles/${MAJOR_VERSION}/binary/${TARGET} IMAGEBUILD && cp -r resources IMAGEBUILD && cp -r etc IMAGEBUILD
	docker build --no-cache --rm -t ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} IMAGEBUILD
	docker tag ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} ${REGISTRY}/${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION}
	rm -rf IMAGEBUILD

shell:
	docker build --rm -t ${BUILD_IMAGE} contrib/builder/binary
	docker run --rm -ti -v $(GOPATH):/go -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} /bin/bash

.PHONY: unit-test build image shell

