## Use slow start to control the cluster traffic distribution

## Introduction

+ The sample project demonstrates how to configure the slow start rule of cluster traffic distribution.

## Preparation

+ A compiled MOSN is needed
```shell
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path
```shell
targetpath=${projectpath}/examples/codes/slow-start-sample/
```

+ Move the target to example code path
```shell
mv main ${targetpath}
cd ${targetpath}
```

## Catalog
```
main        # compiled MOSN
config.json # configure of MOSN
server.go   # mocked http server
```

## Operation instructions

### Start HTTP servers

Start multiple HTTP servers and specify the port of each server (ports 
`8080`, `8081`, `8082` have been configured in `config.json`).

```
go run server.go -port=${port}
```

### Start MOSN

+ Use config to run MOSN.

```
./main start -c config.json
```

### Use CURL for verification

Initially, all servers are started at the same time, slow start has no effect

```shell
$ for i in $(seq 1 12); do curl localhost:2046; echo; done
8080
8081
8082
8081
8082
8082
8080
8081
8082
8081
8082
8082
```

Kill the server with port `8082` and start it after a health check cycle. After the server passing the health check after startup, execute the same command again, and find that the weight of the server with port `8082` is gradually increasing

```shell
$ for i in $(seq 1 12); do curl localhost:2046; echo; done
8082
8081
8080
8081
8082
8081
8080
8081
8082
8081
8080
8081
```