## Run MOSN As envoy-go-extension

## Introduction

+ This example project demonstrates how to configure MOSN as an envoy-go-extension.
+ MOSN listens on a port, and proxy HTTP1 request to an external HTTP server.


## Catelog

```
mosn.json       // MOSN config file
envoy.yaml      // Envoy config file
filter.go       // a simple MOSN filter
build.sh        // Bash script to build MOSN
server.go       // external HTTP server
```

## Instructions

### Build libmosn.so

```
make build
```

This operation will generate libmosn.so

### Start External HTTP Server

```
make run
```

### Start MOSN

```
docker run -d -p 12000:12000 \
    -v `pwd`/libmosn.so:/usr/local/envoy-go-extension/libmosn.so \
    -v `pwd`/envoy.yaml:/etc/envoy/envoy-golang.yaml \
    -v `pwd`/mosn.json:/home/admin/mosn/config/mosn.json \
    mosnio/envoy-go-extension:latest
```

### Send Request With Curl

```
curl -v http://127.0.0.1:12000/
```

The above curl command will receive an HTTP 200 response:

```
* About to connect() to 127.0.0.1 port 12000 (#0)
*   Trying 127.0.0.1...
* Connected to 127.0.0.1 (127.0.0.1) port 12000 (#0)
> GET / HTTP/1.1
> User-Agent: curl/7.29.0
> Host: 127.0.0.1:12000
> Accept: */*
> 
< HTTP/1.1 200 OK
< from: external http server
< date: Mon, 09 Jan 2023 02:03:16 GMT
< content-length: 39
< content-type: text/plain; charset=utf-8
< x-envoy-upstream-service-time: 1
< server: envoy
< 
* Connection #0 to host 127.0.0.1 left intact
response body from external http server
```

On the external HTTP server, we will observe a new http header "Foo: bar"

```
[UPSTREAM]receive request /
Accept -> [*/*]
X-Forwarded-Proto -> [http]
X-Request-Id -> [5c59b8b3-08fb-4baa-90ce-319e88a8f0df]
Foo -> [bar]
X-Envoy-Expected-Rq-Timeout-Ms -> [15000]
User-Agent -> [curl/7.29.0]
```