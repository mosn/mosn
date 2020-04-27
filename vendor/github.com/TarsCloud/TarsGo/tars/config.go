package tars

import (
	"time"

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
	Node     string
	App      string
	Server   string
	LogPath  string
	LogSize  uint64
	LogNum   uint64
	LogLevel string
	Version  string
	LocalIP  string
	BasePath string
	DataPath string
	config   string
	notify   string
	log      string
	Adapters map[string]adapterConfig

	Container   string
	Isdocker    bool
	Enableset   bool
	Setdivision string
	//add server timeout
	AcceptTimeout  time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	HandleTimeout  time.Duration
	IdleTimeout    time.Duration
	ZombileTimeout time.Duration
	QueueCap       int
	//add tcp config
	TCPReadBuffer  int
	TCPWriteBuffer int
	TCPNoDelay     bool
	//add routine number
	MaxInvoke int32
	//add adapter & report config
	PropertyReportInterval time.Duration
	StatReportInterval     time.Duration
	MainLoopTicker         time.Duration
}

type clientConfig struct {
	Locator                 string
	stat                    string
	property                string
	modulename              string
	refreshEndpointInterval int
	reportInterval          int
	AsyncInvokeTimeout      int
	//add client timeout
	ClientQueueLen         int
	ClientIdleTimeout      time.Duration
	ClientReadTimeout      time.Duration
	ClientWriteTimeout     time.Duration
	ReqDefaultTimeout      int32
	ObjQueueMax            int32
	AdapterProxyTicker     time.Duration
	AdapterProxyResetCount int
}
