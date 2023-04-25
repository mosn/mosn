//go:build MOSNTest
// +build MOSNTest

package autoconfig

import (
	"strings"
	"testing"
	"time"

	. "mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
)

func TestUpdateExtend(t *testing.T) {
	Scenario(t, "test update mosn extend config", func() {
		var m *mosn.MosnOperator
		Setup(func() {
			m = mosn.StartMosn(extendConfig, "-f", "auto_config=true")
			Verify(m, NotNil)
			time.Sleep(2 * time.Second) // wait mosn start
		})
		Case("update extend", func() {
			content, err := m.GetMosnConfig(34901, "")
			Verify(err, Equal, nil)
			t := strings.Contains(string(content), "test_extend")
			Verify(t, Equal, true)
			// update
			config := `[
				{
					"type": "new_test",
					"config": {
						"new_test":"new_test"
					}
				}
			]`
			err = m.UpdateConfig(34901, "extend", config)
			Verify(err, Equal, nil)
			// wait auto config dump
			time.Sleep(4 * time.Second)
			mcfg := m.LoadMosnConfig() // read config from files
			Verify(len(mcfg.Extends), Equal, 2)
		})
		TearDown(func() {
			m.Stop()
		})
	})
}

const extendConfig = `{
	"servers":[
		{
			"default_log_path":"stdout"
		}
	],
	"cluster_manager": {},
	"extends": [
		{
			"type": "test_extend",
			"config": {
				 "test":"test"
			}
		}
	],
	"admin": {
		"address": {
			"socket_address": {
				"address": "127.0.0.1",
				"port_value": 34901
			}
		}
	}
}`
