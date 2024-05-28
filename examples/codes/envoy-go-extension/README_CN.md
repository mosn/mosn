## 运行 MOSN envoy-go-extension

## 简介

+ 该样例工程演示了如何配置 MOSN, 将 MOSN 作为 envoy-go-extension 来运行
+ MOSN 监听一个端口，收到 HTTP1 请求后转发给外部 HTTP 服务器


## 目录结构

```
mosn.json       // MOSN 配置
envoy.yaml      // Envoy 配置
filter.go       // 一个简单的 MOSN filter
build.sh        // 用于构建 MOSN 的 Bash 脚本
server.go       // 模拟的外部 Http 服务器
```

## 运行说明

### 构建 libmosn.so

```
make build
```

该操作将产生 libmosn.so 文件


### 启动外部 HTTP 服务器

```
go run server.go
```

### 启动MOSN

```
make run
```

### 使用CURL进行验证

```
curl -v http://127.0.0.1:12000/
```

上述命令将收到 HTTP200 回复：

```
* About to connect() to 127.0.0.1 port 12000 (#0)
*   Trying 127.0.0.1...
* Connected to 127.0.0.1 (127.0.0.1) port 12000 (#0)
> GET / HTTP/1.1
> User-Agent: curl/7.29.0
> Host: 127.0.0.1:12000
> Accept: */*
> 
< HTTP/1.1 200 OK
< from: external http server
< date: Mon, 09 Jan 2023 02:03:16 GMT
< content-length: 39
< content-type: text/plain; charset=utf-8
< x-envoy-upstream-service-time: 1
< server: envoy
< 
* Connection #0 to host 127.0.0.1 left intact
response body from external http server
```

在外部 HTTP 服务器一侧，将观察到由 MOSN filter 添加进请求头的 Foo: bar

```
[UPSTREAM]receive request /
Accept -> [*/*]
X-Forwarded-Proto -> [http]
X-Request-Id -> [5c59b8b3-08fb-4baa-90ce-319e88a8f0df]
Foo -> [bar]
X-Envoy-Expected-Rq-Timeout-Ms -> [15000]
User-Agent -> [curl/7.29.0]
```