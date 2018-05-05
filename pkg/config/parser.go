package config

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"net"
	"strings"

	"time"
)

var logLevelMap = map[string]log.LogLevel{
	"DEBUG": log.DEBUG,
	"FATAL": log.FATAL,
	"ERROR": log.ERROR,
	"WARN":  log.WARN,
	"INFO":  log.INFO,
}

var networkFilterTypeMap = map[string]string{}

var streamFilterTypeMap = map[string]string{}

var clusterTypeMap = map[string]v2.ClusterType{
	"SIMPLE":  v2.SIMPLE_CLUSTER,
	"DYNAMIC": v2.DYNAMIC_CLUSTER,
}

var subClusterTypeMap = map[string]v2.SubClusterType{
	"CONFREG": v2.CONFREG_CLUSTER,
}

var lbTypeMap = map[string]v2.LbType{
	"LB_RANDOM": v2.LB_RANDOM,
}

func ParseLogLevel(level string) log.LogLevel {
	if level != "" {
		if logLevel, ok := logLevelMap[level]; ok {
			return logLevel
		} else {
			log.StartLogger.Fatalln("unsupported log level: ", level)
		}
	}
	//use INFO as default log level
	return log.INFO
}

func ParseServerConfig(c *ServerConfig) *server.Config {
	sc := &server.Config{
		LogPath:         c.DefaultLogPath,
		LogLevel:        ParseLogLevel(c.DefaultLogLevel),
		GracefulTimeout: c.GracefulTimeout.Duration,
		Processor:       c.Processor,
	}

	return sc
}

func ParseProxyFilter(c *FilterConfig) *v2.Proxy {
	proxyConfig := &v2.Proxy{}

	//downstream protocol
	//TODO config(json object) extract and type convert util
	if downstreamProtocol, ok := c.Config["downstream_protocol"]; ok {
		if downstreamProtocol, ok := downstreamProtocol.(string); ok {
			proxyConfig.DownstreamProtocol = downstreamProtocol
		} else {
			log.StartLogger.Fatalln("[downstream_protocol] in proxy filter config is not string")
		}
	} else {
		log.StartLogger.Fatalln("[downstream_protocol] is required in proxy filter config")
	}

	//upstream protocol
	if upstreamProtocol, ok := c.Config["upstream_protocol"]; ok {
		if upstreamProtocol, ok := upstreamProtocol.(string); ok {
			proxyConfig.UpstreamProtocol = upstreamProtocol
		} else {
			log.StartLogger.Fatalln("[upstream_protocol] in proxy filter config is not string")
		}
	} else {
		log.StartLogger.Fatalln("[upstream_protocol] is required in proxy filter config")
	}

	//support dynamic route or not
	if dynamicBool, ok := c.Config["support_dynamic_route"]; ok {
		if dynamicBool, ok := dynamicBool.(string); ok {
			if dynamicBool == "yes" {
				proxyConfig.SupportDynamicRoute = true
			} else if dynamicBool == "no" {
				proxyConfig.SupportDynamicRoute = false
			} else {
				log.StartLogger.Fatalln("[support_dynamic_route] should be \"yes\" or\"not\" ")
			}
		} else {
			log.StartLogger.Fatalln("[support_dynamic_route] in proxy filter support_dynamic_route is not string")
		}
	} else {
		log.StartLogger.Fatalln("[support_dynamic_route] is required in proxy filter config")
	}

	//routes
	if routes, ok := c.Config["routes"]; ok {
		if routes, ok := routes.([]interface{}); ok {
			for _, route := range routes {
				proxyConfig.Routes = append(proxyConfig.Routes, parseRouteConfig(route.(map[string]interface{})))
			}
		} else {
			log.StartLogger.Fatalln("[routes] in proxy filter config is not list of routemap")
		}
	} else {
		log.StartLogger.Fatalln("[routes] is required in proxy filter config")
	}

	return proxyConfig
}

func ParseAccessConfig(c []AccessLogConfig) []v2.AccessLog {
	var logs []v2.AccessLog

	for _, logConfig := range c {
		logs = append(logs, v2.AccessLog{
			Path:   logConfig.LogPath,
			Format: logConfig.LogFormat,
		})
	}

	return logs
}

func parseRouteConfig(config map[string]interface{}) *v2.BasicServiceRoute {
	route := &v2.BasicServiceRoute{}

	//name
	if name, ok := config["name"]; ok {
		if name, ok := name.(string); ok {
			route.Name = name
		} else {
			log.StartLogger.Fatalln("[name] in proxy filter route config is not string")
		}
	} else {
		log.StartLogger.Fatalln("[name] is required in proxy filter route config")
	}

	//service
	if service, ok := config["service"]; ok {
		if service, ok := service.(string); ok {
			route.Service = service
		} else {
			log.StartLogger.Fatalln("[service] in proxy filter route config is not string")
		}
	} else {
		log.StartLogger.Fatalln("[service] is required in proxy filter route config")
	}

	//cluster
	if cluster, ok := config["cluster"]; ok {
		if cluster, ok := cluster.(string); ok {
			route.Cluster = cluster
		} else {
			log.StartLogger.Fatalln("[cluster] in proxy filter route config is not string")
		}
	} else {
		log.StartLogger.Fatalln("[cluster] is required in proxy filter route config")
	}

	return route
}

func ParseFaultInjectFilter(config map[string]interface{}) *v2.FaultInject {

	faultInject := &v2.FaultInject{}

	//percent
	if percent, ok := config["delay_percent"]; ok {
		if percent, ok := percent.(float64); ok {
			faultInject.DelayPercent = uint32(percent)
		} else {
			log.StartLogger.Fatalln("[delay_percent] in fault inject filter config is not integer")
		}
	} else {
		log.StartLogger.Fatalln("[delay_percent] is required in fault inject filter config")
	}

	//duration
	if duration, ok := config["delay_duration"]; ok {
		if duration, ok := duration.(string); ok {
			if duration, error := time.ParseDuration(strings.Trim(duration, `"`)); error == nil {
				faultInject.DelayDuration = uint64(duration)
			} else {
				log.StartLogger.Fatalln("[delay_duration] in fault inject filter config is not valid ,", error)
			}
		} else {
			log.StartLogger.Fatalln("[delay_duration] in fault inject filter config is not a numeric string, like '30s'")
		}
	} else {
		log.StartLogger.Fatalln("[delay_duration] is required in fault inject filter config")
	}

	return faultInject
}

func ParseHealthcheckFilter(config map[string]interface{}) *v2.HealthCheckFilter {
	healthcheck := &v2.HealthCheckFilter{}
	//passthrough
	if passthrough, ok := config["passthrough"]; ok {
		if passthrough, ok := passthrough.(bool); ok {
			healthcheck.PassThrough = passthrough
		} else {
			log.StartLogger.Fatalln("[passthrough] in health check filter config is not bool")
		}
	} else {
		log.StartLogger.Fatalln("[passthrough] is required in healthcheck filter config")
	}

	//cache time
	if cacheTime, ok := config["cache_time"]; ok {
		if cacheTime, ok := cacheTime.(string); ok {
			if duration, error := time.ParseDuration(strings.Trim(cacheTime, `"`)); error == nil {
				healthcheck.CacheTime = duration
			} else {
				log.StartLogger.Fatalln("[cache_time] in health check filter is not valid ,", error)
			}
		} else {
			log.StartLogger.Fatalln("[cache_time] in health check filter config is not a numeric string")
		}
	} else {
		log.StartLogger.Fatalln("[cache_time] is required in healthcheck filter config")
	}

	//cluster_min_healthy_percentages
	if clusterMinHealthyPercentage, ok := config["cluster_min_healthy_percentages"]; ok {
		if clusterMinHealthyPercentage, ok := clusterMinHealthyPercentage.(map[string]interface{}); ok {
			healthcheck.ClusterMinHealthyPercentage = make(map[string]float32)
			for cluster, percent := range clusterMinHealthyPercentage {
				healthcheck.ClusterMinHealthyPercentage[cluster] = float32(percent.(float64))
			}
		} else {
			log.StartLogger.Fatalln("[passthrough] in health check filter config is not bool")
		}
	} else {
		log.StartLogger.Fatalln("[passthrough] is required in healthcheck filter config")
	}
	return healthcheck
}

func ParseListenerConfig(c *ListenerConfig, inheritListeners []*v2.ListenerConfig) *v2.ListenerConfig {
	if c.Name == "" {
		log.StartLogger.Fatalln("[name] is required in listener config")
	}

	if c.Address == "" {
		log.StartLogger.Fatalln("[Address] is required in listener config")
	}

	addr, err := net.ResolveTCPAddr("tcp", c.Address)
	if err != nil {
		log.StartLogger.Fatalln("[Address] not valid:" + c.Address)
	}

	//try inherit legacy listener
	var old *net.TCPListener = nil

	for _, il := range inheritListeners {
		if il.Addr.String() == addr.String() {
			log.StartLogger.Infof("inherit listener addr: %s", c.Address)
			old = il.InheritListener
			il.Remain = true
			break
		}
	}

	return &v2.ListenerConfig{
		Name:                    c.Name,
		Addr:                    addr,
		BindToPort:              c.BindToPort,
		InheritListener:         old,
		PerConnBufferLimitBytes: 1 << 15,
		LogPath:                 c.LogPath,
		LogLevel:                uint8(ParseLogLevel(c.LogLevel)),
		AccessLogs:              ParseAccessConfig(c.AccessLogs),
		DisableConnIo:           c.DisableConnIo,
	}
}

func ParseClusterConfig(c *ClusterConfig) v2.Cluster {
	if c.Name == "" {
		log.StartLogger.Fatalln("[name] is required in cluster config")
	}

	var clusterType v2.ClusterType
	var subclusterType v2.SubClusterType
	if c.Type == "" {
		log.StartLogger.Fatalln("[type] is required in cluster config")
	} else {
		if ct, ok := clusterTypeMap[c.Type]; ok {
			clusterType = ct
			if c.SubType != "" {
				if cs, ok := subClusterTypeMap[c.SubType]; ok {
					subclusterType = cs
				} else {
					log.StartLogger.Fatalln("[unknown sub-cluster type]", c.SubType)
				}
			}
		} else {
			log.StartLogger.Fatalln("unknown cluster type:", c.Type)
		}
	}

	var lbType v2.LbType
	if c.LbType == "" {
		log.StartLogger.Fatalln("[lb_type] is required in cluster config")
	} else {
		if lt, ok := lbTypeMap[c.LbType]; ok {
			lbType = lt
		} else {
			log.StartLogger.Fatalln("unknown lb type:", c.LbType)
		}
	}

	if c.MaxRequestPerConn == 0 {
		c.MaxRequestPerConn = 1024
		log.StartLogger.Infof("[max_request_per_conn] is not specified, use default value 1024")
	}

	if c.ConnBufferLimitBytes == 0 {
		c.ConnBufferLimitBytes = 16 * 1026
		log.StartLogger.Infof("[conn_buffer_limit_bytes] is not specified, use default value 16 * 1026")
	}

	return v2.Cluster{
		Name:                 c.Name,
		ClusterType:          clusterType,
		SubClustetType:       subclusterType,
		LbType:               lbType,
		MaxRequestPerConn:    c.MaxRequestPerConn,
		ConnBufferLimitBytes: c.ConnBufferLimitBytes,
	}
}

func ParseHostConfig(c *ClusterConfig) []v2.Host {
	if c.Hosts == nil || len(c.Hosts) == 0 {
		log.StartLogger.Fatalln("[hosts] is required in cluster config")
	}

	var hosts []v2.Host

	for _, host := range c.Hosts {
		if host.Address == "" {
			log.StartLogger.Fatalln("[host.address] is required in host config")
		}

		hosts = append(hosts, v2.Host{
			Hostname: host.Hostname,
			Address:  host.Address,
			Weight:   host.Weight,
		})
	}

	return hosts
}
