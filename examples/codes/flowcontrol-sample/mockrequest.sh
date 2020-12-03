#!/bin/bash

for i in {1,2,3};do
echo "request count: $i"
curl http://127.0.0.1:2046/test
echo ""
done
