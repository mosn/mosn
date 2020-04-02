## Use SkyWalking as the trace implementation

## Introduction

+ This sample project demonstrates how to configure [SkyWalking](http://skywalking.apache.org/) as a trace implementation of MOSN

## Preparation

+ Install docker & docker-compose

1. [Install docker](https://docs.docker.com/install/) <br>
1. [Install docker-compose](https://docs.docker.com/compose/install/)

+ A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path
``
```
${targetpath} = ${projectpath}/examples/codes/trace/skywalking/http/
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}
```


## Catelog

```
main                           // compiled MOSN
server.go                      // Mocked Http Server
client.go                      // Mocked Http Client
config.json                    // Configure
skywalking-docker-compose.yaml // skywalking docker-compose
```

## Operation instructions

### Start SkyWalking oap & ui
```bash
docker-compose -f skywalking-docker-compose.yaml up -d
```

### Start HTTP Server

```bash
go run server.go
```

### Start MOSN

+ Use config.json to run MOSN.

```bash
./main start -c config.json
```

### Start Http Client
```bash
go run client.go
```

### To view SkyWalking-UI
[http://127.0.0.1:8080](http://127.0.0.1:8080)