## MOSN使用Unix Domain Socket 本机进程间通信
 
## 简介

+ 该样例工程演示了如何配置使得MOSN使用Unix Domain Socket 本机进程间通信

## 准备

+ 需要一个编译好的MOSN程序

```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/uds-sample/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 目录结构

```
main        // 编译完成的MOSN程序
client_config.json // 非TLS的 client 端配置
server_config.json // 非TLS的 server 端配置
uds_server.go // 模拟的Server
rpc_client.go // 模拟的Client
```

## 运行说明

### 启动Server

```
  go run uds_server.go

```

### 启动Server侧MOSN


```
./main start -c server_config.json
```

### 启动 client 端 mosn:

```
./main start -c client_config.json
```

### 启动 Client

```
  go run uds_client.go
```
观察终端显示，间隔打印"echo: Hello, MOSN"


