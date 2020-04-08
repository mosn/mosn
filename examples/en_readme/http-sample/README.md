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
config.json // Configure without TLS
tls.json    // Configure with TLS
```

## Operation instructions

### Start HTTP Server 

```
go run server.go
```

### Start MOSN

+ Use config.json to run MOSN without TLS.

```
./main start -c config.json
```

+ Use tls.json to start SOFAMosb witht TLS.

```
./main start -c tls.json
```


### Use CURL for verification

```
curl http://127.0.0.1:2045/
```
