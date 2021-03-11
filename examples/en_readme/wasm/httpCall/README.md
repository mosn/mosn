## Configure MOSN with Wasm Extension

## Introduction

+ This example project demonstrates how to configure MOSN with Wasm extension.
+ An HTTP request will be made from Wasm extension.
+ Protocol between MOSN is HTTP1.
+ For the convenience of demonstration, MOSN listens to one port. Once receiving an HTTP1 request, MOSN will response with status code 200 directly.

## Preparation

A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ code path

```
${targetpath} = ${projectpath}/examples/codes/wasm/httpCall/
```

+ Move the target to the example code path

```
mv main ${targetpath}/
cd ${targetpath}
```

## Catelog

```
main            // MOSN
config.json     // MOSN config file
filter-go.go    // Wasm source file written in go
filter-c.cc     // Wasm source file written in c
makefile        // makefile to compile wasm source file into wasm extension
server.go       // mocked external http server
```

## Instructions

### Compile Wasm Extension

```
make
```

This operation will generate filter.wasm

### Start MOSN

```
./main start -c config.json
```

### Start external HTTP server
```
go run server.go
```

### Verification

```
curl -v http://127.0.0.1:2045/ -d "haha"
```
Now we should be able to observe wasm-related log printed in MOSN side.

```
[INFO] response header from http://127.0.0.1:2046/: Content-Length: 39
[INFO] response header from http://127.0.0.1:2046/: Content-Type: text/plain; charset=utf-8
[INFO] response header from http://127.0.0.1:2046/: From: external http server
[INFO] response body from http://127.0.0.1:2046/: response body from external http server
```