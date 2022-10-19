#!/usr/bin/sudo /bin/bash

TPROXY_PORT=16000

ip rule add fwmark 1 table 100
ip route add local 0.0.0.0/0 dev lo table 100
iptables -t mangle -I PREROUTING ! -s localhost -p tcp -j TPROXY --on-port $TPROXY_PORT --tproxy-mark 1/1