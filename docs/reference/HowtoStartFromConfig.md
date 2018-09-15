# MOSN 从配置文件启动流程

### 启动方法

```bash
mosn start -c '$CONFIG_FILE'
```
### 启动流程

**1. 配置初始化**

- server config 初始化，如log path、log level。

```go
sc := config.ParseServerConfig(&serverConfig)
```

- network filter 初始化，如 proxy config。

```go
nfcf := getNetworkFilter(serverConfig.NetworkFilters)
```

- stream filters 初始化，如 fault inject。

```go
sfcf := getStreamFilters(serverConfig.StreamFilters)
```

- cluster manager filter初始化。

```go
cmf := &clusterManagerFilter{}
```

**2. 创建服务器**

```go
srv := server.NewServer(sc, nfcf, sfcf, cmf)
```
**3. 添加监听器**

```go
if serverConfig.Listeners == nil || len(serverConfig.Listeners ) == 0 {
	log.Fatal("no listener found")
}

for _, listenerConfig := range serverConfig.Listeners {
	srv.AddListener(config.ParseListenerConfig(&listenerConfig))
}
```
**4. 更新 upstream cluster 信息**

```go
var clusters []v2.Cluster
clusterMap := make(map[string][]v2.Host)

for _, cluster := range c.ClusterManager.Clusters {
	parsed := config.ParseClusterConfig(&cluster)
	clusters = append(clusters, parsed)
	clusterMap[parsed.Name] = config.ParseHostConfig(&cluster)
}
cmf.cccb.UpdateClusterConfig(clusters)

for clusterName, hosts := range clusterMap {
	cmf.chcb.UpdateClusterHost(clusterName, 0, hosts)
}
```
**5. 开启服务**

```go
srv.Start()
```
