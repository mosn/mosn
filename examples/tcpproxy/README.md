## 使用SOFAMesh作为TCP Proxy

## 简介

+ 该样例工程演示了如何配置使得SOFAMesh作TCP Proxy代理
+ SOFAMesh收到一个TCP请求，会根据请求得源地址、目的地址(不配置则为任意地址)转发到对应的cluster

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
main          // 编译完成的SOFAMesh程序
http.go       // 模拟的Http Server
rpc_server.go // 模拟的RPC Server
rpc_client.go // 模拟的RPC Client
config.json   // 非TLS的配置
```

## 运行说明

### 启动SOFAMesh


```
./main start -c config.json
```

+ 转发HTTP

  + 启动一个HTTP Server

  ```
  go run http.go 
  ```

  + 使用CURL进行验证

  ```
  curl http://127.0.0.1:2045/
  ```
+ 转发RPC

  + 启动一个RPC Server

  ```
  go run rpc_server.go
  ```

  + 运行RPC Client 进行验证

  ```
  // 验证一次请求
  go run rpc_client.go
  // 使用-t 让client持续发送请求
  go run rpc_client.go -t
  ```
