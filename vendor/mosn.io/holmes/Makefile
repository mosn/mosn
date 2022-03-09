modules=$(shell go list ./... | grep -v example)
test:
	GO111MODULE=on go test -gcflags "-N -l -v" $(modules)

lint:
	golangci-lint run --timeout=10m --exclude-use-default=false --tests=false --skip-dirs=example