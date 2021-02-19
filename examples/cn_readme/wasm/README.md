## 使用 MOSN 运行 Wasm 扩展

## 简介

+ 该样例工程演示了如何配置 MOSN, 使用 Wasm 扩展来处理 HTTP 请求
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
${targetpath} = ${projectpath}/examples/codes/wasm/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```

## 目录结构

```
main        // 编译完成的MOSN程序
config.json // 非TLS的 mosn 配置
filter.go   // Wasm 扩展程序的源码文件
makefile    // 用于编译 wasm 文件
```

## 运行说明

### 编译 wasm 文件

```
make name=filter
```

该操作将产生 filter.wasm 文件

### 启动MOSN

```
./main start -c config.json
```

### 使用CURL进行验证

```
curl -v http://127.0.0.1:2045/ -d "haha"
```

执行该命令后，返回的 HTTP 响应头中应当包含: Go-Wasm-Header: hello wasm

MOSN 侧将观察到 wasm 扩展打印的相关日志