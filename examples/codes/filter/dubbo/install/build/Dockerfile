FROM istio/proxyv2:1.7.7

COPY build/mosn /usr/local/bin/mosn
# v2
COPY build/mosn_config_dubbo_xds.json /etc/istio/mosn/mosn_config_dubbo_xds.json
# v3
# COPY build/mosn_config_dubbo_xds_v3.json /etc/istio/mosn/mosn_config_dubbo_xds.json
