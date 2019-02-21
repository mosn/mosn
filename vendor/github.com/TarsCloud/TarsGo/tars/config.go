package tars

import (
	"github.com/TarsCloud/TarsGo/tars/util/endpoint"
)

var svrCfg *serverConfig
var cltCfg *clientConfig

// GetServerConfig : Get server config
func GetServerConfig() *serverConfig {
	Init()
	return svrCfg
}

// GetClientConfig : Get client config
func GetClientConfig() *clientConfig {
	Init()
	return cltCfg
}

type adapterConfig struct {
	Endpoint endpoint.Endpoint
	Protocol string
	Obj      string
	Threads  int
}

type serverConfig struct {
	Node      string
	App       string
	Server    string
	LogPath   string
	LogSize   uint64
	LogNum    uint64
	LogLevel  string
	Version   string
	LocalIP   string
	BasePath  string
	DataPath  string
	config    string
	notify    string
	log       string
	netThread int
	Adapters  map[string]adapterConfig

	Container   string
	Isdocker    bool
	Enableset   bool
	Setdivision string
}

type clientConfig struct {
	Locator                 string
	stat                    string
	property                string
	modulename              string
	refreshEndpointInterval int
	reportInterval          int
	AsyncInvokeTimeout      int
}
