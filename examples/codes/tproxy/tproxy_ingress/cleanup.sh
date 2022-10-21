#!/usr/bin/sudo /bin/bash

TPROXY_PORT=16000
LO_MARK=1

ip rule del from all fwmark $LO_MARK lookup 100
ip route del local 0.0.0.0/0 dev lo table 100
iptables -t mangle -D PREROUTING ! -s localhost -p tcp -j TPROXY --on-port $TPROXY_PORT --tproxy-mark $LO_MARK 