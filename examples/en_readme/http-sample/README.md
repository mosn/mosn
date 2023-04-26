## Use MOSN as HTTP Proxy

## Introduction

+ This sample project demonstrates how to configure MOSN as a HTTP Proxy.
+ Protocol between MOSN is HTTP2.
+ For the convenience of demonstration, MOSN listens to two ports, one forwards the client request,
 and one forwards to the server after receiving the request.
 
## Preparation

+ A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/http-sample/
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}

```

## Catelog

```
main        // compiled MOSN
server.go   // Mocked SofaRpc Server
client.go   // Mocked SofaRpc client
client_config.json // Configure without TLS
server_config.json // Configure without TLS
tls_client_config.json    // Configure with TLS
tls_server_config.json    // Configure with TLS
```
> example configs: 📑 [server_config.json](https://github.com/mosn/mosn/blob/master/examples/codes/http-sample/server_config.json) and  📑 [client_config.json](https://github.com/mosn/mosn/blob/master/examples/codes/http-sample/client_config.json)

## Operation instructions

### Start HTTP Server 

```
go run server.go
```

### Start MOSN

+ Use non-TLS configs to run MOSN without TLS.

run client side:
```
./main start -c client_config.json
```

run server side:
```
./main start -c server_config.json
```

+ Use TLS configs to start MOSN witht TLS.

run client side:
```
./main start -c tls_client_config.json
```

run server side:
```
./main start -c tls_server_config.json
```

### Use CURL for verification

```
curl http://127.0.0.1:2045/
```
