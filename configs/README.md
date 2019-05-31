There are some simple config examples for different scenarios 
+ mosn_config.json
  + a simple sofarpc mosn config
+ mosn_http_tls.json
  + a simple http mosn config
+ mosn_autoprotocol.json
  + a simple auto protcol mosn config
+ mosn_iptables.json
  + a simple "transparent proxy" mosn config, the example iptables command is `iptables -t nat -A PREROUTING -p tcp --dport 9080 -j REDIRECT --to-ports 15001`
+ mosn_xprotocol_dubbo.json
  + a simple xprotocol mosn config, sub protocol is dubbo implemented in mosn
