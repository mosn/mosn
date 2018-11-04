#!/usr/bin/env bash

set -e
echo "" > coverage.txt

for d in $(go list ./pkg/...); do
    output="$(go test -v -coverprofile=profile.out -covermode=atomic $d)"
    echo "$output"
    fail=$(echo "$output" | grep "\-\-\- FAIL:" | wc -l)
    if [ $fail -ne 0 ]; then
	    exit 1
    fi
    if [ -f profile.out ]; then
        cat profile.out >> coverage.txt
        rm profile.out
    fi
done
