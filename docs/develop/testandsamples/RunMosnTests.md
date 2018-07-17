# 运行 MOSN proxy 示例

+ 目录 `mosn/pkg/tests/` 中的是将 MOSN 作为转发平面的一些示例程序，每个示例程序均集成了 Client、MOSN、Server
+ 其中，转发拓扑为：Client <---下游协议---> MOSN <---上游协议---> Server
+ 当前，我们支持的上下游协议为 http1.x、http2.0、SOFA 协议族（bolt v1/v2）等
+ 以 bolt2http2 为例，此示例表示，下游协议为bolt，上游协议为http2，其他类似

## 如何运行
+ 仍然以 bolt2http2 为例，你只需要在其目录下，使用如下命令，即可将一个简单而又基本完备的 MOSN 程序跑起来：

```bash
cd bolt2http2
go run proxy.go
```
+ 当你看到如下的输出时，表明 Client -> MOSN -> Server 正向链路是通的

```bash
[UPSTREAM]receive request &{Method:POST
 URL:/ Proto:HTTP/2.0 ProtoMajor:2 ProtoMinor:0 Header:
 map[Accept-Encoding:[gzip] Service:[com.alipay.test.TestService:1.0] 
 Classname:[com.alipay.sofa.rpc.core.request.SofaRequest] Version:[1] 
 ...
```
+ 当你看到如下的输出时，表明 Client <- MOSN <- Server ，即 response 被正确转发
```bash
[CLIENT]Receive data:
**com.alipay.sofa.rpc.core.request.SofaRequestaccept-encodinggzipdateFri,
 GMTservicecom.alipay.test.TestService:1.0
user-agentGo-http-client/2.0content-length695
...
```
## 示例程序简要说明

+ 在 main 函数中有三个 goroutine 分别用于开启 Server、MOSN、Client

### MOSN 开启时的一些配置

+ 如下用于将 MOSN 作为接收后端的 server 进行初始化，logpath 默认输出到 stdout，cm 用于管理cluster
```go
srv := server.NewServer(&server.Config{
    LogPath:  "stdout",
    LogLevel: log.DEBUG,
}, cmf, cm)
```

+ 如下用于添加更新 cluster 以及对应的 host
```go
cmf.cccb.UpdateClusterConfig(clustersrpc())
cmf.chcb.UpdateClusterHost(TestCluster, 0, rpchosts())
```

+ 如下用于给 server 添加监听器：
```go
srv.AddListener(rpcProxyListener(), &proxy.GenericProxyFilterConfigFactory{
    Proxy: genericProxyConfig(),
}, []types.StreamFilterChainFactory{sf, sh})
```
+ 其中 `rpcProxyListener()` 中可修改 MOSN 监听的地址和端口等
+ `GenericProxyFilterConfigFactory` 用于生成 proxy 相关的配置，包括配置上下游监听协议、以及路由信息等
+ `StreamFilterChainFactory` 中可用于配置心跳、故障注入等 stream 级别的信息

