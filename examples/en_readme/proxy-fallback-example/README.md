## Use MOSN as Request Proxy

## Introduction

+ This project shows how to configure MOSN so that it can support auto protocol matching, and fallback to source ip-port automatically
+ When MOSN receives an HTTP1 request, it will auto detect the HTTP1 protocol and forward to upstream http cluster
+ When MOSN receives an request with unknown request, it will forward to source ip-addr through TCP-PROXY

## Preparation

A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/proxy-fallback-sample/
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}
```


## Catelog

```
main          // compiled MOSN
config.json   // Configure without TLS
H1server.go   // Http Server
TcpServer.go  // Tcp Server
```

## instructions

### Start an HTTP Server

```
go run httpserver.go
```

### Start an TCP Server

```
go run tcpserver.go
```

### Start the MOSN

```
./main start -c config.json
```

### Send Request

#### send HTTP1 request
```
curl http://127.0.0.1:2046/
```

#### send tcp request with unknown protocol
```
echo "hello world" | netcat 127.0.0.1 2046
```