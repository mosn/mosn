package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"
)

type FilterConfig struct {
	Type   string
	Config map[string]interface{}
}

type AccessLogConfig struct {
	LogPath   string `json:"log_path"`
	LogFormat string `json:"log_format"`
}

type ListenerConfig struct {
	Name           string
	Address        string
	BindToPort     bool           `json:"bind_port"`
	NetworkFilters []FilterConfig `json:"network_filters"`
	StreamFilters  []FilterConfig `json:"stream_filters"`

	//logger
	LogPath  string `json:"log_path"`
	LogLevel string `json:"log_level"`

	//access log
	AccessLogs []AccessLogConfig `json:"access_logs"`

	// only used in http2 case
	DisableConnIo bool `json:"disable_conn_io"`
}

type ServerConfig struct {
	//default logger
	DefaultLogPath  string `json:"default_log_path"`
	DefaultLogLevel string `json:"default_log_level"`

	//graceful shutdown config
	GracefulTimeout DurationConfig `json:"graceful_timeout"`

	//go processor number
	Processor int

	Listeners []ListenerConfig
}

type HostConfig struct {
	Address  string
	Hostname string
	Weight   uint32
}

type HealthCheckConfig struct {
	Timeout            DurationConfig
	HealthyThreshold   uint32 `json:"healthy_threshold"`
	UnhealthyThreshold uint32 `json:"unhealthy_threshold"`
	Interval           DurationConfig
	IntervalJitter     DurationConfig `json:"interval_jitter"`
	CheckPath          string
	ServiceName        string
}

type ClusterConfig struct {
	Name                 string
	Type                 string
	SubType              string `json:"sub_type"`
	LbType               string `json:"lb_type"`
	MaxRequestPerConn    uint64 `json:"max_request_per_conn"`
	ConnBufferLimitBytes uint32
	HealthCheck          HealthCheckConfig `json:"healthcheck"`
	Hosts                []HostConfig
}

type ClusterManagerConfig struct {
	Clusters []ClusterConfig `json:"clusters"`
}

type MOSNConfig struct {
	Servers        []ServerConfig       `json:"servers"`         //server config
	ClusterManager ClusterManagerConfig `json:"cluster_manager"` //cluster config
	//tracing config
}

//wrapper for time.Duration, so time config can be written in '300ms' or '1h' format
type DurationConfig struct {
	time.Duration
}

func (d *DurationConfig) UnmarshalJSON(b []byte) (err error) {
	d.Duration, err = time.ParseDuration(strings.Trim(string(b), `"`))
	return
}

func (d DurationConfig) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, d.String())), nil
}

func Load(path string) *MOSNConfig {
	log.Println("load config from : ", path)
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalln("load config failed, ", err)
		os.Exit(1)
	}

	var config MOSNConfig
	json.Unmarshal(content, &config)
	return &config
}
