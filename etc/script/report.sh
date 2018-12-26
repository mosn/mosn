#!/usr/bin/env bash

set -e
echo "" > coverage.txt

for d in $(go list ./pkg/...); do
    echo "--------Run test package: $d"
    output="$(go test -v -coverprofile=profile.out -covermode=atomic $d)"
    echo "$output"
    fail=$(echo "$output" | grep "\-\-\- FAIL:" | wc -l)
    echo "--------Finish test package: $d"
    if [ $fail -ne 0 ]; then
	    echo "-------Package $d test FAILED"
	    exit 1
    fi
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
