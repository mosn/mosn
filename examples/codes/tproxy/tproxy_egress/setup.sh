#!/usr/bin/sudo /bin/bash

source ./iptables_config.sh

ip rule add fwmark $mark table 100
ip rule add fwmark $mark table 100
ip route add local 0.0.0.0/0 dev lo table 100

iptables -t mangle -A PREROUTING -p tcp -m mark --mark $mark -j TPROXY --on-port $tproxy_port --tproxy-mark $mark

iptables -t mangle -I OUTPUT -p tcp -o lo -j ACCEPT
iptables -t mangle -A OUTPUT -p tcp -m mark ! --mark 0 -j ACCEPT

for port in ${dports[@]}
    do
    iptables -t mangle -A OUTPUT -p tcp --dport $port -j MARK --set-mark $mark
    done


