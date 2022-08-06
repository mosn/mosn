.PHONY: wasm

wasm:
	docker run -it -v $(shell pwd):/tmp/proxy-wasm-go -e GOPROXY=https://goproxy.cn tinygo/tinygo-dev:a304fb738f2376548e008fbc5c02abf3bdff7156 \
		/bin/bash -c \
		"git clone --depth 1 --branch v0.0.13 https://github.com/tetratelabs/proxy-wasm-go-sdk.git /tmp/proxy-wasm-go/sdk &> /dev/null; \
		cp /tmp/proxy-wasm-go/filter.go /tmp/proxy-wasm-go/sdk; \
		cd /tmp/proxy-wasm-go/sdk; \
		tinygo build -o /tmp/proxy-wasm-go/filter.wasm -scheduler=none -target=wasi -wasm-abi=generic -tags 'abi_010' ./filter.go; \
		rm -rf /tmp/proxy-wasm-go/sdk"
