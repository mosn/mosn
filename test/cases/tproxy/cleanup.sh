#!/usr/bin/sudo /bin/bash

source ./iptables_config.sh

ip rule del from all fwmark $mark lookup 100
ip rule del from all fwmark $mark lookup 100
ip route del local 0.0.0.0/0 dev lo table 100

iptables -t mangle -D PREROUTING -p tcp -m mark --mark $mark -j TPROXY --on-port $tproxy_port --tproxy-mark $mark

iptables -t mangle -D OUTPUT -p tcp -o lo -j ACCEPT
iptables -t mangle -D OUTPUT -p tcp -m mark ! --mark 0 -j ACCEPT

for port in ${dports[@]}
    do
    iptables -t mangle -D OUTPUT -p tcp --dport $port -j MARK --set-mark $mark
    done
