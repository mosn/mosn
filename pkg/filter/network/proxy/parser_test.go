package proxy

import (
	"encoding/json"
	"testing"

	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

func TestParseProxyFilter(t *testing.T) {
	proxyConfigStr := `{
                "name": "proxy",
                "downstream_protocol": "X",
                "upstream_protocol": "Http2",
                "router_config_name":"test_router",
                "extend_config":{
                        "sub_protocol":"example"
                }
        }`
	m := map[string]interface{}{}
	if err := json.Unmarshal([]byte(proxyConfigStr), &m); err != nil {
		t.Error(err)
		return
	}
	proxy, _ := ParseProxyFilter(m)
	if !(proxy.Name == "proxy" &&
		proxy.RouterHandlerName == types.DefaultRouteHandler &&
		proxy.DownstreamProtocol == string(protocol.Xprotocol) &&
		proxy.UpstreamProtocol == string(protocol.HTTP2) && proxy.RouterConfigName == "test_router") {
		t.Error("parse proxy filter failed")
	}
}
