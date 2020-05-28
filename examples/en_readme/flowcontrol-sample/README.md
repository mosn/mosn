## Use MOSN as HTTP Proxy

## Introduction

+ This sample project demonstrates how to configure flow control in MOSN.
 
## Preparation

+ A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/flowcontrol-sample/
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
mockrequest.sh // The scripts which mocks multiple requests for /test

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
bash mockrequest.sh
```

The output should be:

```text
request count: 1
success
request count: 2
blocked
request count: 3
blocked
```
