## 配置标准HTTP协议的SOFAMesh

## 简介

+ 该样例工程演示了如何对Http协议的代理配置限流
+ SOFAMesh之间的协议是HTTP2
+ 为了演示方便，SOFAMesh监听两个端口,一个转发Client的请求，一个收到请求以后转发给Server

## 准备

SOFAMesh 导入限流模块，${projectpath}/cmd/mosn/main/mosn.go里import
```
_ `github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule`
```

编译SOFAMesh程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 将编译好的程序移动到当前目录

```
mv main ${targetpath}/
```

## 目录结构

```
main        // 编译完成的SOFAMesh程序
server.go   // 模拟的Http Server
config.json // 非TLS的配置
client      // 模拟的Http client
```

## 运行说明

### 启动一个HTTP Server

```
go run server.go
```

### 启动SOFAMesh

+ 使用config.json 运行非TLS加密的SOFAMesh

```
./main start -c config.json
```


### 使用Client进行访问

```
go run client.go
```
+ 使用-t 让client持续发送请求 

```
go run client.go -t
```
