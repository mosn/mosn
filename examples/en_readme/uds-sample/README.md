## MOSN uses Unix Domain Socket for native inter-process communication
 
## 简介

+ This sample project demonstrates how to configure MOSN to use Unix Domain Socket native inter-process communication.

## Preparation

A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/udpproxy-sample/
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}

```

## Catelog

```
main        // 编译完成的MOSN程序
client_config.json // 非TLS的 client 端配置
server_config.json // 非TLS的 server 端配置
uds_server.go // 模拟的Server
rpc_client.go // 模拟的Client
```

## Operation instructions

### Start Server

```
  go run uds_server.go

```

### Start Server Side MOSN


```
./main start -c server_config.json
```

### Start Client Side MOSN

```
./main start -c client_config.json
```

### Start Client

```
  go run uds_client.go
```

Print out "echo: Hello, MOSN" in the terminal at intervals
