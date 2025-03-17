## 使用 MOSN 作为 HTTP Ingress 代理(流式响应)

## 简介

+ 该样例工程演示了如何配置使得MOSN作为标准Http协议的 Ingress 代理
+ MOSN之间的协议是HTTP1, 注意配置"http1_use_stream": true
+ 为了演示方便，服务端MOSN监听2048端口（可配置）,收到请求以后转发给Server

## 准备

需要一个编译好的MOSN程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/http-stream-response-sample
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
server_config.json // mosn的 server 端配置
```

## 运行说明

### 启动一个HTTP Server

```
go run server.go
```

### 启动MOSN

启动 server 端:
```
./main start -c server_config.json
```

### 使用CURL进行验证

```
curl http://127.0.0.1:2048/
```
