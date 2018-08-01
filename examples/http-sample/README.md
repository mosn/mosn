## 配置标准HTTP协议的SOFAMesh

## 简介

+ 该样例工程演示了如何配置使得SOFAMesh作为标准Http协议的代理
+ SOFAMesh之间的协议是HTTP2
+ 为了演示方便，SOFAMesh监听两个端口,一个转发Client的请求，一个收到请求以后转发给Server

## 准备

需要一个编译好的SOFAMesh程序
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
tls.json    // TLS配置
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

+ 使用tls.json 开启SOFAMesh之间的TLS加密

```
./main start -c tls.json
```


### 使用CURL进行验证

```
curl http://127.0.0.1:2045/
```
