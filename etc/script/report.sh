#!/usr/bin/env bash

set -e
echo "" > coverage.txt

for d in $(go list ./pkg/...  | grep -v pkg/networkextention); do
    echo "--------Run test package: $d"
    GO111MODULE=on go test -gcflags="all=-N -l" -v -coverprofile=profile.out -covermode=atomic $d
    echo "--------Finish test package: $d"
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done

for d in $(go list ./istio/...); do
    echo "--------Run test package: $d"
    GO111MODULE=on go test -gcflags="all=-N -l" -v -coverprofile=profile.out -covermode=atomic $d
    echo "--------Finish test package: $d"
    if [ -f profile.out ]; then
	cat profile.out >> coverage.txt
	rm profile.out
    fi
done
