## 配置 SOFA RPC 协议 Mesher

## 简介

`sofa-mosn/examples/sofarpc-sample`
样例工程演示了如何配置使得 Mesher 作为 SOFA RPC 代理。Mesher 之间的协议是 HTTP/2。

## 准备

需要一个编译好的 Mesher 程序
```bash
cd ${projectpath}/pkg/mosn
go build
```

将编译好的程序移动到当前目录，目录结构如下 

```bash
mosn  //Mesher程序
server.go //模拟的SofaRpc Server
server.json //SofaRpc Server的Mesher配置
client.json //SofaRpc Client的Mesher配置
client.go //模拟的SofaRpc client
data.go //模拟的SofaRpc数据，用户client的请求和server的响应
```

## 运行说明

### 启动一个HTTP Server

```bash
go run server.go data.go
```

### 启动代理 HTTP Server 的 Mesher

```bash
./mosn start -c server.json
```

### 启动代理HTTP Client的Mesher

```bash
./mosn start -c client.json
```

### 使用Client进行访问

```bash
go run client.go data.go
```
