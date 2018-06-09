package config

import (
	"encoding/json"
	"fmt"
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

//global instance for load & dump
var ConfigPath string
var config MOSNConfig

type FilterConfig struct {
	Type   string                 `json:"type,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

type AccessLogConfig struct {
	LogPath   string `json:"log_path,omitempty"`
	LogFormat string `json:"log_format,omitempty"`
}

type ListenerConfig struct {
	Name           string         `json:"name,omitempty"`
	Address        string         `json:"address,omitempty"`
	BindToPort     bool           `json:"bind_port"`
	NetworkFilters []FilterConfig `json:"network_filters,service_registry"`
	StreamFilters  []FilterConfig `json:"stream_filters,omitempty"`

	//logger
	LogPath  string `json:"log_path,omitempty"`
	LogLevel string `json:"log_level,omitempty"`

	//access log
	AccessLogs []AccessLogConfig `json:"access_logs,omitempty"`

	// only used in http2 case
	DisableConnIo bool `json:"disable_conn_io"`
}

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

type HostConfig struct {
	Address  string `json:"address,omitempty"`
	Hostname string `json:"hostname,omitempty"`
	Weight   uint32 `json:"weight,omitempty"`
}

type ClusterHealthCheckConfig struct {
	Protocol           string         `json:"protocol"`
	Timeout            DurationConfig `json:"timeout"`
	Interval           DurationConfig `json:"interval"`
	IntervalJitter     DurationConfig `json:"interval_jitter"`
	HealthyThreshold   uint32         `json:"healthy_threshold"`
	UnhealthyThreshold uint32         `json:"unhealthy_threshold"`
	CheckPath          string         `json:"check_path,omitempty"`
	ServiceName        string         `json:"service_name,omitempty"`
}

type ClusterSpecConfig struct {
	Subscribes []SubscribeSpecConfig `json:"subscribe,omitempty"`
}

type SubscribeSpecConfig struct {
	ServiceName string `json:"service_name,omitempty"`
}

type ClusterConfig struct {
	Name                 string
	Type                 string
	SubType              string                   `json:"sub_type"`
	LbType               string                   `json:"lb_type"`
	MaxRequestPerConn    uint64                   `json:"max_request_per_conn"`
	ConnBufferLimitBytes uint32                   `json:"conn_buffer_limit_bytes"`
	HealthCheck          ClusterHealthCheckConfig `json:"health_check,omitempty"` //v2.HealthCheck
	ClusterSpecConfig    ClusterSpecConfig        `json:"spec,omitempty"`         //	ClusterSpecConfig
	Hosts                []v2.Host                `json:"hosts,omitempty"`        //v2.Host
}

type ClusterManagerConfig struct {
	AutoDiscovery        bool            `json:"auto_discovery"`
	UseHealthCheckGlobal bool            `json:"use_health_check_global"`
	Clusters             []ClusterConfig `json:"clusters,omitempty"`
}

type ServiceRegistryConfig struct {
	ServiceAppInfo ServiceAppInfoConfig   `json:"application"`
	ServicePubInfo []ServicePubInfoConfig `json:"publish_info,omitempty"`
}

type ServiceAppInfoConfig struct {
	AntShareCloud bool   `json:"ant_share_cloud"`
	DataCenter    string `json:"data_center,omitempty"`
	AppName       string `json:"app_name,omitempty"`
}

type ServicePubInfoConfig struct {
	ServiceName string `json:"service_name,omitempty"`
	PubData     string `json:"pub_data,omitempty"`
}

type MOSNConfig struct {
	Servers         []ServerConfig        `json:"servers,omitempty"`         //server config
	ClusterManager  ClusterManagerConfig  `json:"cluster_manager,omitempty"` //cluster config
	ServiceRegistry ServiceRegistryConfig `json:"service_registry"`          //service registry config, used by service discovery module
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
	ConfigPath, _ = filepath.Abs(path)
	// todo delete
	//ConfigPath = "../../resource/mosn_config_dump_result.json"

	json.Unmarshal(content, &config)
	return &config
}
