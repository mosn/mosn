## Use MOSN as SOFARPC Proxy

## Introduction

+ This sample project demonstrates how to configure MOSN as a SOFARPC Proxy.
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
${targetpath} = ${projectpath}/examples/codes/sofarpc-with-xprotocol-sample/
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

## Operation instructions

### Start Server 

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

### Start Client to send request

```
go run client.go
```
+ Use -t to let client send continue request. 

```
go run client.go -t
```
