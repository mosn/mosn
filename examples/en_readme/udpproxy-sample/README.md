## Use MOSN as TCP Proxy

## Introduction

+ This sample project demonstrates how to configure MOSN as a TCP Proxy.
+ When MOSN receives a TCP request, it will forwards it to the corresponding cluster according to the source address and 
destination address of the request.

## Preparation

A compiled MOSN is needed
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/udpproxy-sample/
```

+ Move the target to example code path

```
mv main ${targetpath}/
cd ${targetpath}

```

## Catelog

```
main          // compiled MOSN
udp.go        // Mocked UDP Server
config.json   // Configure without TLS
```


## Operation instructions

### Start MOSN

```
./main start -c config.json
```

+ Transfer UDP

  + Start UDP Server

  ```
  go run udp.go 
  ```

  + Run nc for verification

  ```
  echo "hello world" | nc -4u localhost 5301
  ```
