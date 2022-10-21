#!/usr/bin/sudo /bin/bash

TPROXY_PORT=16000
LO_MARK=1

ip rule add fwmark $LO_MARK table 100
ip route add local 0.0.0.0/0 dev lo table 100
iptables -t mangle -I PREROUTING -p tcp -j TPROXY --on-port $TPROXY_PORT --tproxy-mark $LO_MARK