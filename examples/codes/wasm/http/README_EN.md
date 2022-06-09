## Configure MOSN with Wasm Extension

## Introduction

+ This example project demonstrates how to configure MOSN with Wasm extension.
+ Protocol between MOSN is HTTP1.
+ For the convenience of demonstration, MOSN listens to one port. Once receiving an HTTP1 request, MOSN will response with status code 200 directly.

## Catelog

```
config.json // MOSN config file
filter.go   // Wasm source file
makefile    // makefile to compile wasm source file into wasm extension
```

## Instructions

### Compile Wasm Extension

```
make
```

This operation will generate filter.wasm

### Pull MOSN image

```
docker pull mosnio/mosn-wasm:v0.21.0
```

### Start MOSN


```
docker run -it --rm -p 2045:2045 -v $(pwd):/etc/wasm/ mosnio/mosn-wasm:v0.21.0
```


### Verification

```
curl -v http://127.0.0.1:2045/ -d "haha"
```

After running this command, the HTTP response should contain a header: Go-Wasm-Header: hello wasm

And we should observe wasm-related log printed in MOSN side.