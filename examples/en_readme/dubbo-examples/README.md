## Use MOSN as Dubbo Proxy

## Introduction

+ This sample project demonstrates how to configure MOSN as a Dubbo Proxy.
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
${targetpath} = ${projectpath}/examples/codes/dubbo/
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}

```


## Catelog

```
main        // compiled MOSN
dubbo-examples/   // the code dir of Dubbo Java provider and consumer
client_config.json // Configure without TLS
server_config.json // Configure without TLS
run.sh     // the scripts of packaging and running Dubbo Java provider and consumer
```

## Operation instructions

### Start Dubbo RPC Provider

```
sh run.sh server
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

### Start Dubbo RPC consumer to send request

```
sh run.sh client
```


