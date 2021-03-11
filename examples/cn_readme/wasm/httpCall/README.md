## 使用 MOSN 运行 Wasm 扩展

## 简介

+ 该样例工程演示了如何配置 MOSN, 使用 Wasm 扩展来处理 HTTP 请求，并从 Wasm 扩展内向外发起 Http 请求
+ MOSN 代理的协议是 HTTP1 协议
+ 为了演示方便，MOSN 监听一个端口，收到 HTTP1 请求后直接返回 200 (status OK)

## 准备

需要一个编译好的MOSN程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/wasm/httpCall/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```

## 目录结构

```
main            // 编译完成的MOSN程序
config.json     // 非TLS的 mosn 配置
filter-go.go    // Wasm 扩展程序的 go 源码文件
filter-c.cc     // Wasm 扩展程序的 c 源码文件
makefile        // 用于编译 wasm 文件
server.go       // 模拟的外部 http 服务器
```

## 运行说明

### 编译 wasm 文件

```
make
```

该操作将产生 filter-go.wasm 文件

### 启动MOSN

```
./main start -c config.json
```

### 启动外部 HTTP 服务器

```
go run server.go
```

### 使用CURL进行验证

```
curl -v http://127.0.0.1:2045/ -d "haha"
```

MOSN 侧将观察到 wasm 扩展打印的相关日志

```
[INFO] response header from http://127.0.0.1:2046/: Content-Length: 39
[INFO] response header from http://127.0.0.1:2046/: Content-Type: text/plain; charset=utf-8
[INFO] response header from http://127.0.0.1:2046/: From: external http server
[INFO] response body from http://127.0.0.1:2046/: response body from external http server
```