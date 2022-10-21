#!/usr/bin/sudo /bin/bash

TPROXY_PORT=16000
LO_MARK=1
TPROXY_MARK=2

ip rule add fwmark $LO_MARK table 100
ip rule add fwmark $TPROXY_MARK table 100
ip route add local 0.0.0.0/0 dev lo table 100

iptables -t mangle -A PREROUTING -p tcp -m mark --mark $LO_MARK -j TPROXY --on-port $TPROXY_PORT --tproxy-mark $TPROXY_MARK

iptables -t mangle -I OUTPUT -p tcp -o lo -j ACCEPT
iptables -t mangle -A OUTPUT -p tcp -m mark ! --mark 0 -j ACCEPT
iptables -t mangle -A OUTPUT -p tcp -j MARK --set-mark $LO_MARK