package tcpproxy

import "testing"

func TestParseTCPProxy(t *testing.T) {
	m := map[string]interface{}{
		"stat_prefix":          "tcp_proxy",
		"cluster":              "cluster",
		"max_connect_attempts": 1000,
		"routes": []interface{}{
			map[string]interface{}{
				"cluster": "test",
				"SourceAddrs": []interface{}{
					map[string]interface{}{
						"address": "127.0.0.1",
						"length":  32,
					},
				},
				"DestinationAddrs": []interface{}{
					map[string]interface{}{
						"address": "127.0.0.1",
						"length":  32,
					},
				},
				"SourcePort":      "8080",
				"DestinationPort": "8080",
			},
		},
	}
	tcpproxy, err := ParseTCPProxy(m)
	if err != nil {
		t.Error(err)
		return
	}
	if len(tcpproxy.Routes) != 1 {
		t.Error("parse tcpproxy failed")
	} else {
		r := tcpproxy.Routes[0]
		if !(r.Cluster == "test" &&
			len(r.SourceAddrs) == 1 &&
			r.SourceAddrs[0].Address == "127.0.0.1" &&
			r.SourceAddrs[0].Length == 32 &&
			len(r.DestinationAddrs) == 1 &&
			r.DestinationAddrs[0].Address == "127.0.0.1" &&
			r.DestinationAddrs[0].Length == 32 &&
			r.SourcePort == "8080" &&
			r.DestinationPort == "8080") {
			t.Error("route failed")
		}
	}
}
