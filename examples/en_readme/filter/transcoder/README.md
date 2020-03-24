## Use MOSN with transcoder from http to bolt

## Introduction

+ This sample project demonstrates how to configure MOSN as a proxy, transcoding Http1 request to Bolt and vice-versa.
+ Client sends Http1 request, Server accept Bolt request; Protocol between MOSN is Bolt.
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
${targetpath} = ${projectpath}/examples/codes/filter/transcoder/
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}

```

## Catelog

```
main        // compiled MOSN
server.go   // Mocked Bolt Server
config.json // Configure 
```

## Operation instructions

### Start Bolt Server 

```
go run server.go
```

### Start MOSN

+ Use config.json to run MOSN.

```
./main start -c config.json
```

### Use CURL for verification

```
curl -v  http://127.0.0.1:2045/hello -H "service:test"
```
