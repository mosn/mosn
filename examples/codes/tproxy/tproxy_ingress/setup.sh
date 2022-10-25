#!/usr/bin/sudo /bin/bash

source ./iptables_config.sh

ip rule add fwmark $mark table 100
ip route add local 0.0.0.0/0 dev lo table 100

for port in ${dports[@]}
    do
    iptables -t mangle -I PREROUTING ! -s localhost -p tcp --dport $port -j TPROXY --on-port $tproxy_port --tproxy-mark $mark 
    done