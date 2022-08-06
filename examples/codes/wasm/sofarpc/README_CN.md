## 使用 MOSN 运行 Wasm 扩展

## 简介

+ 该样例工程演示了如何配置 MOSN, 使用 Wasm 扩展来处理 SofaRPC 请求
+ MOSN 代理的协议是 Bolt 协议
+ 为了演示方便，MOSN 监听一个端口，收到 Bolt 请求后直接返回

## 目录结构

```
config.json // 非TLS的 mosn 配置
filter.go   // Wasm 扩展程序的源码文件
makefile    // 用于编译 wasm 文件
client.go   // 模拟的 SofaRPC client
```

## 运行说明

### 编译 wasm 文件

```
make
```

该操作将产生 filter.wasm 文件

### 获取 MOSN 镜像

```
docker pull mosnio/mosn-wasm:v0.21.0
```

### 启动MOSN

```
docker run -it --rm -p 2045:2045 -v $(pwd):/etc/wasm/ mosnio/mosn-wasm:v0.21.0
```

### 使用 Client 进行访问

```
go run client.go
```

MOSN 侧将观察到 wasm 扩展打印的相关日志, 且协议为 SofaRPC