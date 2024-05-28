#!/bin/bash

mosn=$1
nohup $mosn start -c test/benchmark/mosn_benchmark.json &
nohup /root/java_server/bin/start.sh  >/dev/null
sleep 5s
nohup sofaload -D 10 --qps=2000 -c 200 -t 16 -p sofarpc sofarpc://127.0.0.1:12200