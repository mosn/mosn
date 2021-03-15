## Configure MOSN with Wasm Extension

## Introduction

+ This example project demonstrates how to configure MOSN with Wasm extension to deal with SofaRPC request.
+ Protocol between MOSN is Bolt.
+ For the convenience of demonstration, MOSN listens to one port. Once receiving a Bolt request, MOSN will response directly.

## Preparation

A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ code path

```
${targetpath} = ${projectpath}/examples/codes/wasm/sofarpc
```

+ Move the target to the example code path

```
mv main ${targetpath}/
cd ${targetpath}
```

## Catelog

```
main        // MOSN
config.json // MOSN config file
filter.go   // Wasm source file
makefile    // makefile to compile wasm source file into wasm extension
client.go   // Mocked SofaRPC client
```

## Instructions

### Compile Wasm Extension

```
make name=filter
```

This operation will generate filter.wasm

### Start MOSN

```
./main start -c config.json
```

### Verification

```
go run client.go
```

And we should observe wasm-related log printed in MOSN side.