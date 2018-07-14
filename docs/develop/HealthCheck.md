# 对后端进行主动健康检查
支持对 static cluster 配置 cluster 维度的健康检查能力，配置包括使用的健康检查协议、interval、timeout等

```go
type HealthCheck struct {
   Protocol           string
   Timeout            time.Duration
   Interval           time.Duration
   IntervalJitter     time.Duration
   HealthyThreshold   uint32
   UnhealthyThreshold uint32
   CheckPath          string
   ServiceName        string
}
```

支持对非static cluster配置默认的健康检查能力，包括开关是否打开、使用的默认健康检查协议等，当前支持bolt作为默认的健康检查协议
```go
type ClusterManagerConfig struct {
   AutoDiscovery        bool            `json:"auto_discovery"`
   UseHealthCheckGlobal bool            `json:"use_health_check_global"`
   Clusters             []ClusterConfig `json:"clusters,omitempty"`
}

var DefaultSofaRpcHealthCheckConf = v2.HealthCheck{
   Protocol:           SofaRpc,
   Timeout:            DefaultBoltHeartBeatTimeout,
   HealthyThreshold:   DefaultHealthyThreshold,
   UnhealthyThreshold: DefaultUnhealthyThreshold,
   Interval:           DefaultBoltHeartBeatInterval,
   IntervalJitter:     DefaultIntervalJitter,
}
```

支持对于 cluster 内部所有 hostSet 的机器，单独运行健康检查，并根据健康检查的结果，更新 host 的健康状态


```go
// An upstream cluster (group of hosts).
type Cluster interface {
...

   // set the cluster's health checker
   SetHealthChecker(hc HealthChecker)
   
   // return the cluster's health checker
   HealthChecker() HealthChecker

   OutlierDetector() Detector
}

```
负载均衡算法基于健康检查的主机的健康状态动态选择健康机器

```go
func (l *randomloadbalancer) ChooseHost(context context.Context) types.Host {
...

   hosts := hostset.HealthyHosts()
....
}
```


