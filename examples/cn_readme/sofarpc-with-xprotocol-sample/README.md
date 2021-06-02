## 使用 MOSN 作为 SOFARPC 代理
 
## 简介

+ 该样例工程演示了如何配置使得MOSN作为SofaRpc代理
+ 为了演示方便，MOSN监听两个端口,一个转发Client的请求，一个收到请求以后转发给Server

## 准备

+ 需要一个编译好的MOSN程序

```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/sofarpc-with-xprotocol-sample/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 目录结构

```
main        // 编译完成的MOSN程序
server.go   // 模拟的SofaRpc Server
client.go   // 模拟的SofaRpc client
client_config.json // 非TLS的配置
server_config.json // 非TLS的配置
tls_server_config.json    // TLS配置
tls_client_config.json    // TLS配置
```

## 运行说明

### 启动SOFA RPC Server

```
go run server.go
```

### 启动MOSN

+ 使用普通配置运行非TLS加密的MOSN

启动 client 端:
```
./main start -c client_config.json
```

启动 server 端:
```
./main start -c server_config.json
```

+ 使用TLS配置开启MOSN之间的TLS加密

启动 client 端:
```
./main start -c tls_client_config.json
```

启动 server 端:
```
./main start -c tls_server_config.json
```

### 使用Client进行访问

```
go run client.go
```
+ 使用-t 让client持续发送请求 

```
go run client.go -t
```
