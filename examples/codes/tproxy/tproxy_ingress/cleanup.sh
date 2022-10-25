#!/usr/bin/sudo /bin/bash

source ./iptables_config.sh

ip rule del from all fwmark $mark lookup 100
ip route del local 0.0.0.0/0 dev lo table 100

for port in ${dports[@]}
    do
    iptables -t mangle -D PREROUTING ! -s localhost -p tcp --dport $port -j TPROXY --on-port $tproxy_port --tproxy-mark $mark 
    done