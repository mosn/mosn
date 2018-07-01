SHELL = /bin/bash

TARGET       = mosnd
CONFIG_FILE  = mosn_config.json
GIT_USER     = afe
PROJECT_NAME = gitlab.alipay-inc.com/${GIT_USER}/mosn

MAJOR_VERSION = $(shell cat VERSION)
GIT_VERSION   = $(shell git log -1 --pretty=format:%h)
GIT_NOTES     = $(shell git log -1 --oneline)

BUILD_IMAGE   = godep-builder

IMAGE_NAME = ${GIT_USER}/mosnd
REGISTRY = acs-reg.alipay.com

RPM_BUILD_IMAGE = afenp-rpm-builder
RPM_VERSION     = $(shell cat VERSION | tr -d '-')
RPM_TAR_NAME    = afe-${TARGET}
RPM_SRC_DIR     = ${RPM_TAR_NAME}-${RPM_VERSION}
RPM_TAR_FILE    = ${RPM_SRC_DIR}.tar.gz

ut-local:
	go test ./...

unit-test:
	docker build --rm -t ${BUILD_IMAGE} contrib/builder/binary
	docker run --rm -v $(GOPATH):/go -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make ut-local

build:
	docker build --rm -t ${BUILD_IMAGE} contrib/builder/binary
	docker run --rm -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make build-local

build-host:
	docker build --rm -t ${BUILD_IMAGE} contrib/builder/binary
	docker run --net=host --rm -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} make build-local

binary: build

binary-host: build-host

build-local:
	@rm -rf bundles/${MAJOR_VERSION}/binary
	CGO_ENABLED=0 go build\
		-ldflags "-B 0x$(shell head -c20 /dev/urandom|od -An -tx1|tr -d ' \n') -X main.Version=${MAJOR_VERSION}(${GIT_VERSION})" \
		-v -o ${TARGET} \
		${PROJECT_NAME}/pkg/mosn
	mkdir -p bundles/${MAJOR_VERSION}/binary
	mv ${TARGET} bundles/${MAJOR_VERSION}/binary
	@cd bundles/${MAJOR_VERSION}/binary && $(shell which md5sum) -b ${TARGET} | cut -d' ' -f1  > ${TARGET}.md5
	cp resource/${CONFIG_FILE} bundles/${MAJOR_VERSION}/binary

image:
	@rm -rf IMAGEBUILD
	cp -r contrib/builder/image IMAGEBUILD && cp bundles/${MAJOR_VERSION}/binary/${TARGET} IMAGEBUILD && cp -r resource IMAGEBUILD && cp -r etc IMAGEBUILD
	docker build --no-cache --rm -t ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} IMAGEBUILD
	docker tag ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION} ${REGISTRY}/${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION}
	rm -rf IMAGEBUILD

rpm:
	@sleep 1  # sometimes device-mapper complains for a relax
	docker build --rm -t ${RPM_BUILD_IMAGE} contrib/builder/rpm
	docker run --rm -w /opt/${TARGET}     \
		-v $(shell pwd):/opt/${TARGET}    \
		-e RPM_GIT_VERSION=${GIT_VERSION} \
		-e "RPM_GIT_NOTES=${GIT_NOTES}"   \
	   	${RPM_BUILD_IMAGE} make rpm-build-local

rpm-build-local:
	@rm -rf bundles/${MAJOR_VERSION}/rpm
	mkdir -p bundles/${MAJOR_VERSION}/rpm/${RPM_SRC_DIR}
	cp -r bundles/${MAJOR_VERSION}/binary/${TARGET} bundles/${MAJOR_VERSION}/rpm/${RPM_SRC_DIR}
	cp -r bundles/${MAJOR_VERSION}/binary/${CONFIG_FILE} bundles/${MAJOR_VERSION}/rpm/${RPM_SRC_DIR}
	cp contrib/builder/rpm/${TARGET}.spec bundles/${MAJOR_VERSION}/rpm
	cp contrib/builder/rpm/${TARGET}.service bundles/${MAJOR_VERSION}/rpm/${RPM_SRC_DIR}
	cp contrib/builder/rpm/${TARGET}.logrotate bundles/${MAJOR_VERSION}/rpm/${RPM_SRC_DIR}
	cd bundles/${MAJOR_VERSION}/rpm && tar zcvf ${RPM_TAR_FILE} ${RPM_SRC_DIR}
	mv bundles/${MAJOR_VERSION}/rpm/${RPM_TAR_FILE} ~/rpmbuild/SOURCES
	chown -R root:root ~/rpmbuild/SOURCES/${RPM_TAR_FILE}
	rpmbuild -bb --clean contrib/builder/rpm/${TARGET}.spec             	\
			--define "AFENP_NAME       ${RPM_TAR_NAME}" 					\
			--define "AFENP_VERSION    ${RPM_VERSION}"       		    	\
			--define "AFENP_RELEASE    ${RPM_GIT_VERSION}"     				\
			--define "AFENP_GIT_NOTES '${RPM_GIT_NOTES}'"
	cp ~/rpmbuild/RPMS/x86_64/*.rpm bundles/${MAJOR_VERSION}/rpm
	rm -rf bundles/${MAJOR_VERSION}/rpm/${RPM_SRC_DIR} bundles/${MAJOR_VERSION}/rpm/${TARGET}.spec

shell:
	docker build --rm -t ${BUILD_IMAGE} contrib/builder/binary
	docker run --rm -ti -v $(GOPATH):/go -v $(shell pwd):/go/src/${PROJECT_NAME} -w /go/src/${PROJECT_NAME} ${BUILD_IMAGE} /bin/bash

.PHONY: unit-test build image rpm upload shell
