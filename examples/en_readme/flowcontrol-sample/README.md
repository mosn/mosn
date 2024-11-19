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
client_config.json // Configure without TLS
server_config.json // Configure without TLS
tls_client_config.json    // Configure with TLS
tls_server_config.json    // Configure with TLS
mockrequest.sh // The scripts which mocks multiple requests for /test

```

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

### Dynamic Data Source

if want to use config center as source of flow control rules, refer to dynamic-data-source/README.md