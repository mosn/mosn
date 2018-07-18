## 配置标准HTTP协议的Mesher

## 简介

该样例工程演示了如何配置使得Mesher作为标准Http协议的代理
 Mesher之间的协议是HTTP2

## 准备

需要一个编译好的Mesher程序
```
cd ${projectpath}/cmd/mosn
go build
```

将编译好的程序移动到当前目录，目录结构如下 

```
mosn  //Mesher程序
server.go //HTTP Server
server.json //HTTP Server的Mesher配置
client.json //HTTP Client的Mesher配置
```

## 运行说明

### 启动一个HTTP Server

```
go run server.go
```

### 启动代理HTTP Server的Mesher

```
./mosn start -c server.json
```

### 启动代理HTTP Client的Mesher

```
./mosn start -c client.json
```

### 使用CURL进行验证

+ 按照默认的配置设置，HTTP Server监听本地8080端口，HTTP Client代理监听本地2046端口
+ Mesher代理配置转发请求为Header中包含"service:com.alipay.test.TestService:1.0"

```
//直接访问HTTP Sever，观察现象
curl http://127.0.0.1:8080
//能收到HTTP Server返回的结果
curl --header "service:com.alipay.test.TestService:1.0" http://127.0.0.1:2046
//不能收到HTTP Server返回的结果(其实是返回了404 Not Found）
curl http://127.0.0.1:2046
```

+ 可以按照说明修改配置，进行不同的测试与验证
