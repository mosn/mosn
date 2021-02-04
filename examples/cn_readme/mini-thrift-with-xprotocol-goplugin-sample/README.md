# 使用 goplugin 作为 xprotocol 的编解码器
 
## 简介

+ 该样例工程以Thrift协议的一个子集实现为例，演示了如何开发基于goplugin技术的xprotocol编解码器
+ 该样例工程演示了如何配置MOSN，使其可以加载并使用基于goplugin的xprotocol编解码器

## 开发编解码器
开发基于go-plugin的编解码器，其实质是实现mosn.io/api中所定义的xprotocol框架接口：

```go
type XProtocol interface {
	Protocol

	Heartbeater

	Hijacker

	PoolMode() PoolMode // configure this to use which connpool

	EnableWorkerPool() bool // same meaning as EnableWorkerPool in types.StreamConnection

	// generate a request id for stream to combine stream request && response
	// use connection param as base
	GenerateRequestID(*uint64) uint64
}
```

并实现一个加载编解码器的方法：
```go
func LoadCodec() api.XProtocolCodec
```

方法名可以由开发者在配置文件中自行定义，也可以使用缺省方法名`LoadCodec`。

具体代码，见：
${projectpath}/examples/codes/mini-thrift-with-xprotocol-goplugin-sample/thrift/codec/*


## demo演示
这里我们使用一个mosn作为thrift demo server的sidecar，用另一个demo client访问mosn，用以演示其可行性。

### 1. 准备Thrift的Python3环境

```
sudo pip3 install thrift
```

### 2. 编译Thrift demo应用、Thrift编解码器和MOSN

```bash
➜ cd ${projectpath}/examples/codes/mini-thrift-with-xprotocol-goplugin-sample/
➜ 
➜ 
➜ ./make.sh
==> Making goplugin for thrift protocol...
/Users/dingfei/Go/src/github.com/fdingiit/mosn/examples/codes/mini-thrift-with-xprotocol-goplugin-sample
==> Making thrift demo app...
/Users/dingfei/Go/src/github.com/fdingiit/mosn/examples/codes/mini-thrift-with-xprotocol-goplugin-sample
==> Making mosn...
```

如果一切正常，应该会得到三个编译产物：
- mosn：位于${projectpath}/examples/codes/mini-thrift-with-xprotocol-goplugin-sample/mosn/mosn
- thrift.so：位于${projectpath}/examples/codes/mini-thrift-with-xprotocol-goplugin-sample/thrift/codec/thrift.so
- gen-py目录：位于${projectpath}/examples/codes/mini-thrift-with-xprotocol-goplugin-sample/thrift/tutorial/gen-py

### 3. 配置编解码器信息
- 在config.json中给MOSN配置编解码器信息：
```json
"third_part_codec": {
"codecs": [
  {
    "enable": true,
    "type": "go-plugin",
    "path": "../thrift/codec/thrift.so"
  }
]
}
```

- 在config.json中配置thrift协议的监听器：
```json
"listeners": [
{
  "name": "serverListener",
  "address": "127.0.0.1:3399",
  "bind_port": true,
  "filter_chains": [
    {
      "tls_context": {
        "status": false,
        "ca_cert": "../certs/ca.pem",
        "cert_chain": "../certs/cert.pem",
        "private_key": "../certs/key.pem",
        "verify_client": true,
        "require_client_cert": true
      },
      "filters": [
        {
          "type": "proxy",
          "config": {
            "downstream_protocol": "X",
            "upstream_protocol": "X",
            "extend_config": {
              "sub_protocol": "thrift"
            },
            "router_config_name": "server_router"
          }
        }
      ]
    }
  ],
  "stream_filters": [
    {
      "type": "healthcheck",
      "config": {
      }
    }
  ]
}
]
```

- 在config.json中配置后段server
```json
"cluster_manager": {
"clusters": [
  {
    "name": "serverCluster",
    "type": "SIMPLE",
    "lb_type": "LB_RANDOM",
    "max_request_per_conn": 1024,
    "conn_buffer_limit_bytes": 32768,
    "hosts": [
      {
        "address": "127.0.0.1:9090"
      }
    ]
  }
]
}
```

示例配置文件已包含以上内容。

### 4. 启动MOSN
```bash
➜ cd ${projectpath}/examples/codes/mini-thrift-with-xprotocol-goplugin-sample/
➜ 
➜ 
➜ ./start_mosn.sh
==> Starting mosn...
2021-02-04 20:08:11,37 [INFO] register a new handler maker, name is default, is default: true
2021-02-04 20:08:11,38 [INFO] [config] processor added to configParsedCBMaps
2021-02-04 20:08:11,48 [INFO] [network] [ register pool factory] register protocol: Http1 factory
2021-02-04 20:08:11,48 [INFO] [network] [ register pool factory] register protocol: Http2 factory
2021-02-04 20:08:11,48 [INFO] [network] [ register pool factory] register protocol: X factory
2021-02-04 20:08:11,54 [INFO] load config from :  config.json
2021-02-04 20:08:11,56 [INFO] [mosn] [start] xds service type must be sidecar or router
2021-02-04 20:08:11,56 [INFO] [mosn] [init tracing] disable tracing
2021-02-04 20:08:11,902 [INFO] [mosn] [init codec] loading protocol [thrift] from third part codec
2021-02-04 20:08:11,902 [INFO] [mosn] [init codec] load go plugin codec succeed: ../thrift/codec/thrift.so
2021-02-04 20:08:11,902 [INFO] [mosn] [NewMosn] new mosn created
2021-02-04 20:08:11,903 [INFO] [cluster] [cluster manager] [AddOrUpdatePrimaryCluster] cluster serverCluster updated
2021-02-04 20:08:11,903 [INFO] [upstream] [host set] update host, final host total: 1
2021-02-04 20:08:11,903 [INFO] [cluster] [primaryCluster] [UpdateHosts] cluster serverCluster update hosts: 1
2021-02-04 20:08:11,903 [INFO] parsing listen config:tcp
2021-02-04 20:08:11,903 [INFO] [server] [conn handler] [add listener] add listener: 127.0.0.1:3399
2021-02-04 20:08:11,903 [ERROR] [streamfilter] get stream filter failed, type: healthcheck, error: unsupported stream filter type: healthcheck
2021-02-04 20:08:11,903 [WARN] [streamfilter] createStreamFilterFactoryFromConfig return nil factories
2021-02-04 20:08:11,903 [INFO] [streamfilter] AddOrUpdateStreamFilterConfig add filter chain key: serverListener
2021-02-04 20:08:11,903 [INFO] [router] [virtualhost] [addRouteBase] add a new route rule
2021-02-04 20:08:11,903 [INFO] [router] [routers] [NewRouters] add route matcher default virtual host
2021-02-04 20:08:11,903 [INFO] [router] [routers_manager] [AddOrUpdateRouters] add router: server_router
2021-02-04 20:08:11,903 [INFO] mosn start xds client
2021-02-04 20:08:11,903 [WARN] [feature gate] feature auto_config is not enabled
2021-02-04 20:08:11,903 [WARN] [feature gate] feature XdsMtlsEnable is not enabled
2021-02-04 20:08:11,903 [WARN] [feature ga2021-02-04 20:08:11,903 [INFO] xds client start
2021-02-04 20:08:11,903 [ERROR] StaticResources is null
2021-02-04 20:08:11,903 [WARN] fail to init xds config, skip xds: null point exception
te] feature PayLoadLimitEnable is not enabled
2021-02-04 20:08:11,903 [WARN] [feature gate] feature MultiTenantMode is not enabled
2021-02-04 20:08:11,903 [INFO] mosn parse extend config
2021-02-04 20:08:11,903 [INFO] mosn prepare for start
2021-02-04 20:08:11,903 [INFO] [admin store] [add service] add server Mosn Admin Server
2021-02-04 20:08:11,904 [INFO] [admin store] [mosn state] state changed to 1
2021-02-04 20:08:11,904 [INFO] mosn start server
2021-02-04 20:08:11,904 [INFO] [admin store] [start service] start service Mosn Admin Server on [::]:34902
```

注意下面的log内容，表明MOSN成功加载了thrift编解码器，并在3399端口监听thrift协议：
```bash
2021-02-04 20:08:11,902 [INFO] [mosn] [init codec] load go plugin codec succeed: ../thrift/codec/thrift.so
2021-02-04 20:08:11,902 [INFO] [mosn] [NewMosn] new mosn created
2021-02-04 20:08:11,903 [INFO] [cluster] [cluster manager] [AddOrUpdatePrimaryCluster] cluster serverCluster updated
2021-02-04 20:08:11,903 [INFO] [upstream] [host set] update host, final host total: 1
2021-02-04 20:08:11,903 [INFO] [cluster] [primaryCluster] [UpdateHosts] cluster serverCluster update hosts: 1
2021-02-04 20:08:11,903 [INFO] parsing listen config:tcp
2021-02-04 20:08:11,903 [INFO] [server] [conn handler] [add listener] add listener: 127.0.0.1:3399
```

### 5. 启动demo server
```bash
➜ cd ${projectpath}/examples/codes/mini-thrift-with-xprotocol-goplugin-sample/
➜ 
➜ 
➜ ./start_server.sh
==> Starting thrift demo server...
Starting the server at 0.0.0.0:9090
```

demo server会监听9090端口的请求

### 6. 启动demo client
```bash
➜ cd ${projectpath}/examples/codes/mini-thrift-with-xprotocol-goplugin-sample/
➜ 
➜ 
➜ ./start_client.sh
==> Starting thrift demo client...
================================================
sending request to 127.0.0.1:3399
1+1=2
InvalidOperation: InvalidOperation(whatOp=4, why=u'Cannot divide by 0')
15-10=5
Check log: 5
```

client默认会向本地3399端口（即mosn监听的端口）发送thrift请求。观察client端log，可以看到成功请求了server，并收到了预期内的返回：
```
sending request to 127.0.0.1:3399
1+1=2
InvalidOperation: InvalidOperation(whatOp=4, why=u'Cannot divide by 0')
15-10=5
Check log: 5
```

观察server端log，可以看到对应的请求日志，以及第一个请求之后mosn持续向server发的心跳检测包：
```
add(1,1)
calculate(1, Work(comment=None, num1=1, num2=0, op=4))
calculate(1, Work(comment=None, num1=15, num2=10, op=2))
getStruct(1)
('2021.02.04-20:20:13', 'ping()')
('2021.02.04-20:20:28', 'ping()')
('2021.02.04-20:20:43', 'ping()')
('2021.02.04-20:20:58', 'ping()')
('2021.02.04-20:21:13', 'ping()')
('2021.02.04-20:21:28', 'ping()')
('2021.02.04-20:21:43', 'ping()')
('2021.02.04-20:21:58', 'ping()')
```

观察codec的log：${projectpath}/examples/codes/mini-thrift-with-xprotocol-goplugin-sample/mosn/thrift-codec.log，可以看到codec打印的一系列日志：
```
...
...
2021/02/04 20:20:13 in encodeRequest
2021/02/04 20:20:13 in decodeResponse
2021/02/04 20:20:28 in encodeRequest
2021/02/04 20:20:28 in decodeResponse
2021/02/04 20:20:43 in encodeRequest
2021/02/04 20:20:43 in decodeResponse
2021/02/04 20:20:58 in encodeRequest
2021/02/04 20:20:58 in decodeResponse
2021/02/04 20:21:13 in encodeRequest
2021/02/04 20:21:13 in decodeResponse
2021/02/04 20:21:28 in encodeRequest
2021/02/04 20:21:28 in decodeResponse
2021/02/04 20:21:43 in encodeRequest
2021/02/04 20:21:43 in decodeResponse
2021/02/04 20:21:58 in encodeRequest
...
...
```



