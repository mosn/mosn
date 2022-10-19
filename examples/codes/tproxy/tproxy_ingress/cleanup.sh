#!/usr/bin/sudo /bin/bash

TPROXY_PORT=16000

ip rule del from all fwmark 0x1 lookup 100
ip route del local 0.0.0.0/0 dev lo table 100
iptables -t mangle -D PREROUTING ! -s localhost -p tcp -j TPROXY --on-port $TPROXY_PORT --tproxy-mark 1/1