package lib

import (
	"encoding/json"
	"time"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/test/framework"
	"mosn.io/mosn/test/lib/mosn"
	"mosn.io/mosn/test/lib/types"
)

type Config struct {
	Protocol string      `json:"protocol"`
	Config   interface{} `json:"config"`
}

// InitMosn simplify MOSN test case with test framework
// Most cases can use the functions instead of `Setup` and `TearDown`
// This function also needs to be called in framework.Scenario
func InitMosn(mosnConfigStr string, srvConfigs ...*Config) (m *mosn.MosnOperator, servers []types.MockServer) {
	servers = make([]types.MockServer, len(srvConfigs))
	framework.Setup(func() {
		m = mosn.StartMosn(mosnConfigStr)
		framework.Verify(m, framework.NotNil)
		for idx, s := range srvConfigs {
			srv := CreateServer(s.Protocol, s.Config)
			framework.Verify(srv, framework.NotNil) // should not be nil
			servers[idx] = srv
			go srv.Start()
		}
		time.Sleep(time.Duration(len(srvConfigs)+1) * time.Second) // wait mosn and server start
	})
	framework.TearDown(func() {
		log.DefaultLogger.Infof("finish case")
		m.Stop()
		for _, srv := range servers {
			srv.Stop()
		}
	})
	return
}

// CreateConfig translate config string to Config
func CreateConfig(str string) *Config {
	s := &Config{}
	if err := json.Unmarshal([]byte(str), s); err != nil {
		panic(err)
	}
	return s
}
