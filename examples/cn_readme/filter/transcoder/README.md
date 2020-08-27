## 使用 MOSN 实现协议报文转换

## 简介

+ 该样例工程演示了如何配置使得 MOSN 可以将您的协议报文转为为指定目标形式
+ 客户端发起 Http1 请求，服务端接受 Bolt 请求；Mosn之间的协议是 Bolt
+ 为了演示方便，MOSN 监听两个端口,一个转发Client的请求，一个收到请求以后转发给Server

## 准备

需要一个编译好的 Mosn 程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/filter/transcoder
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 目录结构

```
main        // 编译完成的 Mosn 程序
server.go   // 模拟的 Bolt Server
client_config.json // client 端示例配置
server_config.json // server 端示例配置
```

## 运行说明

### 启动一个 Bolt Server

```
go run server.go
```

### 启动 MOSN

+ 使用配置文件运行 MOSN

启动 client 端:
```
./main start -c client_config.json
```

启动 server 端:
```
./main start -c server_config.json
```

### 使用CURL进行验证

```
curl -v  http://127.0.0.1:2045/hello -H "service:test"
```
