lint:
	gometalinter ./ | grep -v ALL_CAPS|grep -v underscores|grep -v unhandled|grep -v 'not checked' |grep -v unsafe | grep -v unused

lint2:
	golangci-lint run
test:
	go test
