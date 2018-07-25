# 配置文件说明

## 配置文件示例

+ [示例](Configfile.json)
+ 配置结构体

MOSN 配置文件主要由如下 `MOSNConfig` 结构体中的成员组成
```go
type MOSNConfig struct {
	Servers         []ServerConfig        `json:"servers,omitempty"`         //server config
	ClusterManager  ClusterManagerConfig  `json:"cluster_manager,omitempty"` //cluster config
	ServiceRegistry ServiceRegistryConfig `json:"service_registry"`          //service registry config, used by service discovery module
	//tracing config
	RawDynamicResources json.RawMessage `json:"dynamic_resources,omitempty"` //dynamic_resources raw message
	RawStaticResources  json.RawMessage `json:"static_resources,omitempty"`  //static_resources raw message
}
```   
## ServerConfig 配置块

参考 [示例](Configfile.json) 中的 `servers` 块，其对应的结构体为 `ServerConfig`
包含启动 MOSN 作为 Server 的一些配置项

```go
type ServerConfig struct {
	//default logger
	DefaultLogPath  string `json:"default_log_path,omitempty"`
	DefaultLogLevel string `json:"default_log_level,omitempty"`

	//graceful shutdown config
	GracefulTimeout DurationConfig `json:"graceful_timeout"`

	//go processor number
	Processor int

	Listeners []ListenerConfig `json:"listeners,omitempty"`
}
``` 
+ `DefaultLog*` 等定义当前 Server 块默认的日志路径

+ `ListenerConfig` 对应 Server 的监听器对象，结构体为

```go
type ListenerConfig struct {
	Name          string         `json:"name,omitempty"`
	Address       string         `json:"address,omitempty"`
	BindToPort    bool           `json:"bind_port"`
	FilterChains  []FilterChain  `json:"filter_chains"`
	StreamFilters []FilterConfig `json:"stream_filters,omitempty"`

	//logger
	LogPath  string `json:"log_path,omitempty"`
	LogLevel string `json:"log_level,omitempty"`

	//HandOffRestoredDestinationConnections
	HandOffRestoredDestinationConnections bool `json:"handoff_restoreddestination"`

	//access log
	AccessLogs []AccessLogConfig `json:"access_logs,omitempty"`

}

```

1. `BindToPort` 需要设置为 true , 否则监听器将不工作
2. `FilterConfig` 为定义的 stream filters, 当前支持 fault_inject 和 healthcheck
    + 其结构为: 
    ```go
    type FilterConfig struct {
        Type   string
        Config map[string]interface{}
    }
    ```

    + 示例:
    ```json
    {
        "type": "fault_inject", 
        "config": {
            "delay_percent": 100, 
            "delay_duration_ms": 2000
        }
    }
    ```
3. `FilterChain` 用于配置 Proxy 等，在 FilterConfig 的基础上包了一层,
    + 结构为：
    ```go
    type FilterChain struct {
        FilterChainMatch string         `json:"match,omitempty"`
        TLS              TLSConfig      `json:"tls_context,omitempty"`
        Filters          []FilterConfig `json:"filters"`
    }
    ```
    FilterConfig 定义了 proxy 具体参考

## Upstream 配置块

参考 [示例](Configfile.json) 中的 `clusters` 块，其对应的结构体为 `ClusterConfig`，
定义了 MOSN 上游的 Cluster 以及 Host 信息

```go
type ClusterConfig struct {
	Name                 string
	Type                 string
	SubType              string `json:"sub_type"`
	LbType               string `json:"lb_type"`
	MaxRequestPerConn    uint32
	ConnBufferLimitBytes uint32
	CircuitBreakers      []*CircuitBreakerdConfig `json:"circuit_breakers"`
	HealthCheck          ClusterHealthCheckConfig `json:"health_check,omitempty"` //v2.HealthCheck
	ClusterSpecConfig    ClusterSpecConfig        `json:"spec,omitempty"`         //	ClusterSpecConfig
	Hosts                []v2.Host                `json:"hosts,omitempty"`        //v2.Host
	LBSubsetConfig       v2.LBSubsetConfig
	TLS                  TLSConfig `json:"tls_context,omitempty"`
}
```
+ `CircuitBreakers` 为熔断的配置项
+ `HealthCheck` 定义了对此 cluster 做健康检查的配置
+ `LBSubsetConfig` 定义了此 cluster 的 subset 信息
+ `Hosts` 为 cluster 中具体的 host ，结构体定义为

```go
type Host struct {
	Address  string
	Hostname string
	Weight   uint32
	MetaData Metadata
}
```