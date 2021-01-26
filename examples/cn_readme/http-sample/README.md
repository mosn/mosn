## 使用 MOSN 作为 HTTP 代理

## 简介

+ 该样例工程演示了如何配置使得MOSN作为标准Http协议的代理
+ MOSN之间的协议是HTTP2
+ 为了演示方便，MOSN监听两个端口,一个转发Client的请求，一个收到请求以后转发给Server

## 准备

需要一个编译好的MOSN程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/http-sample/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 目录结构

```
main        // 编译完成的MOSN程序
server.go   // 模拟的Http Server
client_config.json // 非TLS的 client 端配置
server_config.json // 非TLS的 server 端配置
tls_server_config.json    // TLS server 配置
tls_client_config.json    // TLS client 配置
```

## 运行说明

### 启动一个HTTP Server

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

### 使用CURL进行验证

```
curl http://127.0.0.1:2045/
```
