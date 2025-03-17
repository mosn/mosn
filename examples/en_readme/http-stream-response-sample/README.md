## Use MOSN as HTTP Ingress Proxy（stream response）

## Introduction

+ This sample project demonstrates how to configure MOSN as a HTTP Ingress Proxy.
+ Protocol between MOSN is HTTP1. Pay attention to configuring 'http1_use_stream': true
+ For the convenience of demonstration, the server MOSN listens on port 2048 (configurable) and forwards the request to the server after receiving it
 
## Preparation

+ A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/http-stream-response-sample
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}

```

## Catelog

```
main        // compiled MOSN
server.go   // Mocked http Server
server_config.json // mosn server configure 
```

## Operation instructions

### Start HTTP Server 

```
go run server.go
```

### Start MOSN

run server side:
```
./main start -c server_config.json
```

### Use CURL for verification

```
curl http://127.0.0.1:2048/
```
