package config

import (
	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/server"
	"log"
	"strings"
	"time"
	log2 "gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

var logLevelMap = map[string]log2.LogLevel{
	"DEBUG":log2.DEBUG,
	"FATAL": log2.FATAL,
	"ERROR": log2.ERROR,
	"WARN": log2.WARN,
	"INFO": log2.INFO,
}



func ParseServerConfig(c *ServerConfig) *server.Config {
	sc := &server.Config{}

	if c.AccessLog != ""{
		sc.LogPath = c.AccessLog
	}

	if c.LogLevel != ""{
		if logLevel, ok := logLevelMap[c.LogLevel]; ok{
			sc.LogLevel = logLevel
		}else{
			log.Fatalln("unknown log level:" + c.LogLevel)
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
			log.Fatal("[downstream_protocol] in proxy filter config is not string")
		}
	} else {
		log.Fatal("[downstream_protocol] is required in proxy filter config")
	}

	//upstream protocol
	if upstreamProtocol, ok := c.Config["upstream_protocol"]; ok {
		if upstreamProtocol, ok := upstreamProtocol.(string); ok {
			proxyConfig.UpstreamProtocol = upstreamProtocol
		} else {
			log.Fatal("[upstream_protocol] in proxy filter config is not string")
		}
	} else {
		log.Fatal("[upstream_protocol] is required in proxy filter config")
	}

	//routes
	if routes, ok := c.Config["routes"]; ok {
		if routes, ok := routes.([]interface{}); ok {
			for _, route := range routes {
				proxyConfig.Routes = append(proxyConfig.Routes, parseRouteConfig(route.(map[string]interface{})))
			}
		} else {
			log.Fatal("[routes] in proxy filter config is not list of routemap")
		}
	} else {
		log.Fatal("[routes] is required in proxy filter config")
	}

	//todo add accesslogs
	return proxyConfig
}

func parseRouteConfig(config map[string]interface{}) *v2.BasicServiceRoute {
	route := &v2.BasicServiceRoute{}

	//name
	if name, ok := config["name"]; ok {
		if name, ok := name.(string); ok {
			route.Name = name
		} else {
			log.Fatal("[name] in proxy filter route config is not string")
		}
	} else {
		log.Fatal("[name] is required in proxy filter route config")
	}

	//service
	if service, ok := config["service"]; ok {
		if service, ok := service.(string); ok {
			route.Service = service
		} else {
			log.Fatal("[service] in proxy filter route config is not string")
		}
	} else {
		log.Fatal("[service] is required in proxy filter route config")
	}

	//cluster
	if cluster, ok := config["cluster"]; ok {
		if cluster, ok := cluster.(string); ok {
			route.Cluster = cluster
		} else {
			log.Fatal("[cluster] in proxy filter route config is not string")
		}
	} else {
		log.Fatal("[cluster] is required in proxy filter route config")
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
			log.Fatal("[delay_percent] in fault inject filter config is not integer")
		}
	} else {
		log.Fatal("[delay_percent] is required in fault inject filter config")
	}

	//duration
	if duration, ok := config["delay_duration_ms"]; ok {
		if duration, ok := duration.(float64); ok {
			faultInject.DelayDuration = uint64(duration)
		} else {
			log.Fatal("[delay_duration_ms] in fault inject filter config is not integer")
		}
	} else {
		log.Fatal("[delay_duration_ms] is required in fault inject filter config")
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
			log.Fatal("[passthrough] in health check filter config is not bool")
		}
	} else {
		log.Fatal("[passthrough] is required in healthcheck filter config")
	}

	//cache time
	if cacheTime, ok := config["cache_time_ms"]; ok {
		if cacheTime, ok := cacheTime.(float64); ok {
			healthcheck.CacheTime = time.Duration(cacheTime)
		} else {
			log.Fatal("[cache_time_ms] in health check filter config is not integer")
		}
	} else {
		log.Fatal("[cache_time_ms] is required in healthcheck filter config")
	}

	//cluster_min_healthy_percentages
	if clusterMinHealthyPercentage, ok := config["cluster_min_healthy_percentages"]; ok {
		if clusterMinHealthyPercentage, ok := clusterMinHealthyPercentage.(map[string]interface{}); ok {
			healthcheck.ClusterMinHealthyPercentage = make(map[string]float32)
			for cluster, percent := range clusterMinHealthyPercentage {
				healthcheck.ClusterMinHealthyPercentage[cluster] = float32(percent.(float64))
			}
		} else {
			log.Fatal("[passthrough] in health check filter config is not bool")
		}
	} else {
		log.Fatal("[passthrough] is required in healthcheck filter config")
	}
	return healthcheck
}

func ParseListenerConfig(c *ListenerConfig) *v2.ListenerConfig {
	if c.Name == "" {
		log.Fatal("[name] is required in listener config")
	}

	if c.Address == "" {
		log.Fatal("[Address] is required in listener config")
	}

	return &v2.ListenerConfig{
		Name:                 c.Name,
		Addr:                 c.Address,
		BindToPort:           c.BindToPort,
		PerConnBufferLimitBytes: 1024 * 32,
	}
}

func ParseClusterConfig(c *ClusterConfig) v2.Cluster {
	if c.Name == "" {
		log.Fatal("[name] is required in cluster config")
	}

	if c.Type == "" {
		log.Fatal("[type] is required in cluster config")
	}

	if c.LbType == "" {
		log.Fatal("[lb_type] is required in cluster config")
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
		ClusterType:          v2.ClusterType(strings.ToUpper(c.Type)),
		LbType:               v2.LbType(strings.ToUpper(c.LbType)),
		MaxRequestPerConn:    c.MaxRequestPerConn,
		ConnBufferLimitBytes: c.ConnBufferLimitBytes,
	}
}

func ParseHostConfig(c *ClusterConfig) []v2.Host {
	if c.Hosts == nil || len(c.Hosts) == 0 {
		log.Fatal("[hosts] is required in cluster config")
	}

	var hosts []v2.Host

	for _, host := range c.Hosts {
		if host.Address == "" {
			log.Fatal("[host.address] is required in host config")
		}

		hosts = append(hosts, v2.Host{
			Hostname: host.Hostname,
			Address:  host.Address,
			Weight:   host.Weight,
		})
	}

	return hosts
}
