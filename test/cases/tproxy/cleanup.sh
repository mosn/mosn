#!/usr/bin/sudo /bin/bash

TPROXY_PORT=16000
LO_MARK=1
TPROXY_MARK=2

ip rule del from all fwmark $LO_MARK lookup 100
ip rule del from all fwmark $TPROXY_MARK lookup 100
ip route del local 0.0.0.0/0 dev lo table 100

iptables -t mangle -D PREROUTING -p tcp -m mark --mark $LO_MARK -j TPROXY --on-port $TPROXY_PORT --tproxy-mark $TPROXY_MARK

iptables -t mangle -D OUTPUT -p tcp -o lo -j ACCEPT
iptables -t mangle -D OUTPUT -p tcp -m mark ! --mark 0 -j ACCEPT
iptables -t mangle -D OUTPUT -p tcp -j MARK --set-mark $LO_MARK