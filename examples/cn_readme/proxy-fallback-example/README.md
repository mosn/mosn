## 使用 MOSN 作为请求代理

## 简介

+ 该样例工程演示了如何配置使得 MOSN 支持自动协议识别，以及在自动识别失败时，根据请求的源地址和端口进行转发
+ MOSN 收到 HTTP1 请求时，会自动识别 HTTP1 协议，向上游 HTTP cluster 转发
+ MOSN 收到未能识别协议的请求时，会通过 TCP-PROXY 转发到对应的 cluster

## 准备

需要一个编译好的MOSN程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/proxy-fallback-sample/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 目录结构

```
main          // 编译完成的MOSN程序
config.json   // 非TLS的配置
H1server.go   // 模拟的 Http Server
TcpServer.go  // 模拟的 Tcp Server
```

## 运行说明

### 启动一个 HTTP Server

```
go run httpserver.go
```

### 启动一个 TCP Server

```
go run tcpserver.go
```

### 启动 MOSN

```
./main start -c config.json
```

### 请求验证

```
curl http://127.0.0.1:2046/
echo "hello world" | netcat 127.0.0.1 2046
```