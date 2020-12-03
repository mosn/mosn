## Use MOSN as TCP Proxy

## Introduction

+ This sample project demonstrates how to configure MOSN as a TCP Proxy.
+ When MOSN receives a TCP request, it will forwards it to the corresponding cluster according to the source address and 
destination address of the request.

## Preparation

A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/tcpproxy-sample/
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}

```

## Catelog

```
main          // compiled MOSN
http.go       // Mocked Http Server
rpc_server.go // Mocked RPC Server
rpc_client.go // Mocked RPC Client
config.json   // Configure without TLS
```


## Operation instructions

### Start MOSN

```
./main start -c config.json
```

+ Transfer HTTP

  + Start HTTP Server

  ```
  go run http.go 
  ```

  + Run curl for verification

  ```
  curl http://127.0.0.1:2045/
  ```
+ Transfer RPC

  + Start RPC Server

  ```
  go run rpc_server.go
  ```

  + Run RPC Client for verification

  ```
  // Verify an request
  go run rpc_client.go
  // Use -t for client sending continuous request 
  go run rpc_client.go -t
  ```
