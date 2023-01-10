.PHONY: build build-local run

build-local:
	rm -rf build && mkdir build
	cp ../../../cmd/moe/main/* filter.go build
	cd build && GO111MODULE=on CGO_ENABLED=1 go build --buildmode=c-shared -v -o libmosn.so && mv libmosn.so ../libmosn.so
	rm -rf build

build:
	docker run --platform linux/amd64 --net=host --rm \
		-v $(GOPATH)/src/mosn.io/mosn:/go/src/mosn.io/mosn \
		-w /go/src/mosn.io/mosn/examples/codes/envoy-go-extension \
		golang:1.18 make build-local

run:
	docker run --platform linux/amd64 -p 12000:12000 \
		-v $(PWD)/libmosn.so:/usr/local/envoy-go-extension/libmosn.so \
		-v $(PWD)/envoy.yaml:/etc/envoy/envoy-golang.yaml \
		-v $(PWD)/mosn.json:/home/admin/mosn/config/mosn.json \
		--add-host=host.docker.internal:host-gateway \
        mosnio/envoy-go-extension:latest
