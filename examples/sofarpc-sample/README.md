## 配置SOFARPC协议Mesher

## 简介

该样例工程演示了如何配置使得Mesher作为SofaRpc代理
Mesher之间的协议是HTTP2

## 准备

需要一个编译好的Mesher程序
```
cd ${projectpath}/pkg/mosn
go build
```

将编译好的程序移动到当前目录，目录结构如下 

```
mosn  //Mesher程序
server.go //模拟的SofaRpc Server
server.json //SofaRpc Server的Mesher配置
client.json //SofaRpc Client的Mesher配置
client.go //模拟的SofaRpc client
data.go //模拟的SofaRpc数据，用户client的请求和server的响应
```

## 运行说明

### 启动一个HTTP Server

```
go run server.go data.go
```

### 启动代理HTTP Server的Mesher

```
./mosn start -c server.json
```

### 启动代理HTTP Client的Mesher

```
./mosn start -c client.json
```

### 使用Client进行访问

```
go run client.go data.go
```
