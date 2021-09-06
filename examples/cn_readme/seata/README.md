## 使用 Seata 协调分布式事务

## 简介

+ 该样例工程演示了如何配置 Seata filter 与 Seata TC 交互对分布式事务进行协调。
+ 该 Seata filter 仅对 http1 协议进行拦截，配合 seata 完成对分布式事务的协调工作。
+ 被协调的 http rest 接口服务，try、confirm、cancel 对应的接口请求参数应一致。
+ 被协调的 http rest 接口服务，try、confirm、cancel 对应的接口必须是 POST 接口。
+ 需要业务方传递 xid（全局事务id）到事务分支接口，通过 request header 传递。

## 配置说明

+ 下面的配置见 `examples/codes/seata/server_a/service_a_config.json`：
```
"stream_filters": [
	{
		"type": "seata",
		"config": {
			"addressing": "service-a",
			"serverAddressing": "localhost:8091",
			"commitRetryCount": 5,
			"rollbackRetryCount": 5,
			"transactionInfos": [
				{
					"requestPath": "/service-a/begin",
					"timeout": 60000
				}
			]
		}
	}
]
```
1. addressing 为被代理的服务的唯一标识，可以是 applicationID，也可以是 k8s 中 service name.
2. serverAddressing 为 seata tc server 的访问地址，在 k8s 中，可以配置为 ${FQDN}:{service 端口}。
3. transactionInfos 配置了要开启全局事务的接口。通过 requestPath 与接口 url 匹配，匹配成功则 mosn 会与 
seata tc server 交互开启全局事务。timeout 单位为毫秒，用来标识全局事务的超时时间。

+ 下面的配置见 `examples/codes/seata/server_b/service_b_config.json`：
```
"stream_filters": [
	{
		"type": "seata",
		"config": {
			"addressing": "service-b",
			"serverAddressing": "localhost:8091",
			"commitRetryCount": 5,
			"rollbackRetryCount": 5,
			"tccResources": [
				{
					"prepareRequestPath": "/service-b/try",
					"commitRequestPath": "/service-b/confirm",
					"rollbackRequestPath": "/service-b/cancel"
				}
			]
		}
	}
]
```
tccResources 配置了 TCC 分支事务对应的接口。如果请求 url 与 `prepareRequestPath` 匹配，并且 
requestHeader 中存在 key `x_seata_xid`，则 mosn 将向 seata tc server 注册分支事务。当全
局事务提交时，seata tc server 会通知 mosn 提交分支事务，mosn 将自动调用 `commitRequestPath`
对应的接口。全局回滚时，seata tc server 会通知 mosn 回滚分支事务，mosn 将自动调用 
`rollbackRequestPath` 对应的接口来回滚。

## 准备

需要一个编译好的 MOSN 程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 需要运行 seata tc server，地址：https://github.com/opentrx/seata-golang/tree/v2


+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/seata/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```

## 目录结构

```
main        // 编译完成的 MOSN 程序
|-- server_a
|-- |-- server_a.go 
|-- |-- server_a_config.json 
|-- server_b
|-- |-- server_b.go 
|-- |-- server_b_config.json 
|-- server_c
|-- |-- server_c.go 
|-- |-- server_c_config.json 
```

## 运行说明

### 启动 seata tc server

参考 https://github.com/opentrx/seata-golang/tree/v2

### 启动 MOSN

+ 使用 server_a_config.json 启动 MOSN

```
./main start -c server_a/server_a_config.json
```

+ 使用 server_b_config.json 启动 MOSN

```
./main start -c server_b/server_b_config.json
```

+ 使用 server_c_config.json 启动 MOSN

```
./main start -c server_c/server_c_config.json
```

### 启动 rest 服务

+ 启动 server_a.go
+ 启动 server_b.go
+ 启动 server_c.go

### 验证 Seata 的处理

+ 浏览器访问 http://localhost:2046/service-a/begin, tc 会自动调用 server_b、server_c 的 confirm 接口
