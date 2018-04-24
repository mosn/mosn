package config

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	log2 "gitlab.alipay-inc.com/afe/mosn/pkg/log"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"log"
	"net"
	"time"
)

var logLevelMap = map[string]log2.LogLevel{
	"DEBUG": log2.DEBUG,
	"FATAL": log2.FATAL,
	"ERROR": log2.ERROR,
	"WARN":  log2.WARN,
	"INFO":  log2.INFO,
}

var networkFilterTypeMap = map[string]string{}

var streamFilterTypeMap = map[string]string{}

var clusterTypeMap = map[string]v2.ClusterType{
	"SIMPLE": v2.SIMPLE_CLUSTER,
}

var lbTypeMap = map[string]v2.LbType{
	"LB_RANDOM": v2.LB_RANDOM,
}

func ParseServerConfig(c *ServerConfig) *server.Config {
	sc := &server.Config{DisableConnIo: c.DisableConnIo}
	if c.LoggerPath != "" {
		sc.LogPath = c.LoggerPath
	}

	if c.LoggerLevel != "" {
		if logLevel, ok := logLevelMap[c.LoggerLevel]; ok {
			sc.LogLevel = logLevel
		} else {
			log.Fatalln("unknown log level:" + c.LoggerLevel)
		}
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
			log.Fatalln("[downstream_protocol] in proxy filter config is not string")
		}
	} else {
		log.Fatalln("[downstream_protocol] is required in proxy filter config")
	}

	//upstream protocol
	if upstreamProtocol, ok := c.Config["upstream_protocol"]; ok {
		if upstreamProtocol, ok := upstreamProtocol.(string); ok {
			proxyConfig.UpstreamProtocol = upstreamProtocol
		} else {
			log.Fatalln("[upstream_protocol] in proxy filter config is not string")
		}
	} else {
		log.Fatalln("[upstream_protocol] is required in proxy filter config")
	}

	//routes
	if routes, ok := c.Config["routes"]; ok {
		if routes, ok := routes.([]interface{}); ok {
			for _, route := range routes {
				proxyConfig.Routes = append(proxyConfig.Routes, parseRouteConfig(route.(map[string]interface{})))
			}
		} else {
			log.Fatalln("[routes] in proxy filter config is not list of routemap")
		}
	} else {
		log.Fatalln("[routes] is required in proxy filter config")
	}

	//accesslog
	if accesslog, ok := c.Config["accesslog"]; ok {
		if accesslogs, ok := accesslog.([]interface{}); ok {
			for _, alog := range accesslogs {
				proxyConfig.AccessLogs = append(proxyConfig.AccessLogs, parseAccessConfig(alog.(map[string]interface{})))
			}
		} else {
			log.Fatalln("[accesslog] in proxy filter config is not list of acclessmap")
		}
	} else {
		log.Fatalln("[accesslog] is required in proxy filter config")
	}


	//todo add accesslogs
	return proxyConfig
}

func parseAccessConfig(config map[string]interface{}) *v2.AccessLog {
	accesslog := &v2.AccessLog{}

	//accesslog path
	if alpath, ok := config["accesslogPath"]; ok {
		if pathstr, ok := alpath.(string); ok {
			accesslog.Path = pathstr
		} else {
			log.Fatalln("[accesslogPath] in proxy filter accesslog config is not string")
		}
	} else {
		log.Fatalln("[accesslogPath] is required in proxy filter accesslog config")
	}

	//accesslog format
	if alformat, ok := config["accesslogFormat"]; ok {
		if formatstr, ok := alformat.(string); ok {
			accesslog.Format = formatstr
		} else {
			log.Fatalln("[accesslogFormat] in proxy filter accesslog config is not string")
		}
	} else {
		log.Fatalln("[accesslogFormat] is required in proxy filter accesslog config")
	}

	return accesslog
}

func parseRouteConfig(config map[string]interface{}) *v2.BasicServiceRoute {
	route := &v2.BasicServiceRoute{}

	//name
	if name, ok := config["name"]; ok {
		if name, ok := name.(string); ok {
			route.Name = name
		} else {
			log.Fatalln("[name] in proxy filter route config is not string")
		}
	} else {
		log.Fatalln("[name] is required in proxy filter route config")
	}

	//service
	if service, ok := config["service"]; ok {
		if service, ok := service.(string); ok {
			route.Service = service
		} else {
			log.Fatalln("[service] in proxy filter route config is not string")
		}
	} else {
		log.Fatalln("[service] is required in proxy filter route config")
	}

	//cluster
	if cluster, ok := config["cluster"]; ok {
		if cluster, ok := cluster.(string); ok {
			route.Cluster = cluster
		} else {
			log.Fatalln("[cluster] in proxy filter route config is not string")
		}
	} else {
		log.Fatalln("[cluster] is required in proxy filter route config")
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
			log.Fatalln("[delay_percent] in fault inject filter config is not integer")
		}
	} else {
		log.Fatalln("[delay_percent] is required in fault inject filter config")
	}

	//duration
	if duration, ok := config["delay_duration_ms"]; ok {
		if duration, ok := duration.(float64); ok {
			faultInject.DelayDuration = uint64(duration)
		} else {
			log.Fatalln("[delay_duration_ms] in fault inject filter config is not integer")
		}
	} else {
		log.Fatalln("[delay_duration_ms] is required in fault inject filter config")
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
			log.Fatalln("[passthrough] in health check filter config is not bool")
		}
	} else {
		log.Fatalln("[passthrough] is required in healthcheck filter config")
	}

	//cache time
	if cacheTime, ok := config["cache_time_ms"]; ok {
		if cacheTime, ok := cacheTime.(float64); ok {
			healthcheck.CacheTime = time.Duration(cacheTime)
		} else {
			log.Fatalln("[cache_time_ms] in health check filter config is not integer")
		}
	} else {
		log.Fatalln("[cache_time_ms] is required in healthcheck filter config")
	}

	//cluster_min_healthy_percentages
	if clusterMinHealthyPercentage, ok := config["cluster_min_healthy_percentages"]; ok {
		if clusterMinHealthyPercentage, ok := clusterMinHealthyPercentage.(map[string]interface{}); ok {
			healthcheck.ClusterMinHealthyPercentage = make(map[string]float32)
			for cluster, percent := range clusterMinHealthyPercentage {
				healthcheck.ClusterMinHealthyPercentage[cluster] = float32(percent.(float64))
			}
		} else {
			log.Fatalln("[passthrough] in health check filter config is not bool")
		}
	} else {
		log.Fatalln("[passthrough] is required in healthcheck filter config")
	}
	return healthcheck
}

func ParseListenerConfig(c *ListenerConfig, inheritListeners []*v2.ListenerConfig) *v2.ListenerConfig {
	if c.Name == "" {
		log.Fatalln("[name] is required in listener config")
	}

	if c.Address == "" {
		log.Fatalln("[Address] is required in listener config")
	}

	addr, err := net.ResolveTCPAddr("tcp", c.Address)
	if err != nil {
		log.Fatalln("[Address] not valid:" + c.Address)
	}

	//try inherit legacy listener
	var old *net.TCPListener = nil

	for _, il := range inheritListeners {
		if il.Addr.String() == addr.String() {
			log.Println("inherit listener addr:", c.Address)
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
	}
}

func ParseClusterConfig(c *ClusterConfig) v2.Cluster {
	if c.Name == "" {
		log.Fatalln("[name] is required in cluster config")
	}

	var clusterType v2.ClusterType
	if c.Type == "" {
		log.Fatalln("[type] is required in cluster config")
	} else {
		if ct, ok := clusterTypeMap[c.Type]; ok {
			clusterType = ct
		} else {
			log.Fatalln("unknown cluster type:", c.Type)
		}
	}

	var lbType v2.LbType
	if c.LbType == "" {
		log.Fatalln("[lb_type] is required in cluster config")
	} else {
		if lt, ok := lbTypeMap[c.LbType]; ok {
			lbType = lt
		} else {
			log.Fatalln("unknown lb type:", c.LbType)
		}
	}

	if c.MaxRequestPerConn == 0 {
		c.MaxRequestPerConn = 1024
		log.Println("[max_request_per_conn] is not specified, use default value 1024")
	}

	if c.ConnBufferLimitBytes == 0 {
		c.ConnBufferLimitBytes = 16 * 1026
		log.Println("[conn_buffer_limit_bytes] is not specified, use default value 16 * 1026")
	}

	return v2.Cluster{
		Name:                 c.Name,
		ClusterType:          clusterType,
		LbType:               lbType,
		MaxRequestPerConn:    c.MaxRequestPerConn,
		ConnBufferLimitBytes: c.ConnBufferLimitBytes,
	}
}

func ParseHostConfig(c *ClusterConfig) []v2.Host {
	if c.Hosts == nil || len(c.Hosts) == 0 {
		log.Fatalln("[hosts] is required in cluster config")
	}

	var hosts []v2.Host

	for _, host := range c.Hosts {
		if host.Address == "" {
			log.Fatalln("[host.address] is required in host config")
		}

		hosts = append(hosts, v2.Host{
			Hostname: host.Hostname,
			Address:  host.Address,
			Weight:   host.Weight,
		})
	}

	return hosts
}
