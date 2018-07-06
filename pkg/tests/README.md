##文件说明

+ 当前目录是mosn作为转发平面的一些事例程序，
每个事例程序均集成了client、mosn、server
+ 其中，转发拓扑为：Client <---下游协议---> MOSN <---上游协议---> Server
+ 当前，我们支持的上下游协议为http1.x, http2.0, sofa协议族 (bolt v1/v2,TR )等,
+ 以bolt2http2为例, 此事例表示，下游协议为bolt，上游协议为http2，
其他类似

##如何运行
+ 仍然以bolt2http2为例，你只需要在其目录下，使用如下命令，即可将一个简单而又基本完备的mosn
程序跑起来：

```
cd bolt2http2
go run proxy.go
```
+ 当你看到如下的输出时，表明 clint -> mosh -> server 正向链路是通的

```
[UPSTREAM]receive request &{Method:POST
 URL:/ Proto:HTTP/2.0 ProtoMajor:2 ProtoMinor:0 Header:
 map[Accept-Encoding:[gzip] Service:[com.alipay.test.TestService:1.0] 
 Classname:[com.alipay.sofa.rpc.core.request.SofaRequest] Version:[1] 
 ...
 
```
+ 当你看到如下的输出时，表明 clint <- mosh <- server ，即response被正确转发
```
[CLIENT]Receive data:
�,��com.alipay.sofa.rpc.core.request.SofaRequestaccept-encodinggzipdateFri, 06 Jul 2018 13:21:53
 GMTservicecom.alipay.test.TestService:1.0
user-agentGo-http-client/2.0content-length695
...
```
## 事例程序简要说明

+ 在 main函数中有三个goroutine分别用于开启server、mosn、client

### 这里主要介绍下mosn开启时的一些配置

+ 如下用于将mosh作为接收后端的server进行初始化，logpath默认输出到stdout，cm用于管理cluster
```go
srv := server.NewServer(&server.Config{
    LogPath:  "stdout",
    LogLevel: log.DEBUG,
}, cmf, cm)
 
```
+ 如下用于添加更新cluster以及对应的host
```go
cmf.cccb.UpdateClusterConfig(clustersrpc())
cmf.chcb.UpdateClusterHost(TestCluster, 0, rpchosts())
```

+ 如下用于给server添加监听器：
```go
srv.AddListener(rpcProxyListener(), &proxy.GenericProxyFilterConfigFactory{
    Proxy: genericProxyConfig(),
}, []types.StreamFilterChainFactory{sf, sh})

//其中rpcProxyListener()中可修改mosn监听的地址和端口等，
// GenericProxyFilterConfigFactory用于生成proxy相关的配置，包括配置上下游监听协议、以及路由信息等，
// StreamFilterChainFactory中可用于配置心跳、故障注入等stream级别的信息
```
