## Use MOSN as UDP Proxy

## Introduction

+ This sample project demonstrates how to configure MOSN as a UDP Proxy.
+ When MOSN receives a UDP request, it will forwards it to the corresponding cluster according to the source address and 
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

+ Start MOSN

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
  
+ Output

  The client side will receive what it sent out：
  
  ```
  ~ tiny$ echo "hello world" | nc -4u localhost 5301
  hello world
  ```
  
  The server side will receive what client sent out:
  
  ```
  udpproxy-sample tiny$ go run udp.go
  Listening on udp port 5300 ...
  Receive from 127.0.0.1:55373, len:12, data:hello world
  ```
  
+ Mosn debug logs

  An UDP session will be created when receiving from a new remote addr and no associated session found.
  And it will be closed after 5 idle checks without data transmission on this session.
  So read timeout debug log will be printed, which is as expected.
  ```
  2020-08-10 12:13:18,315 [INFO] [network] [read loop] do read err: read udp 127.0.0.1:55373->127.0.0.1:5300: i/o timeout
  2020-08-10 12:13:19,320 [INFO] [network] [read loop] do read err: read udp 127.0.0.1:55373->127.0.0.1:5300: i/o timeout
  2020-08-10 12:13:20,325 [INFO] [network] [read loop] do read err: read udp 127.0.0.1:55373->127.0.0.1:5300: i/o timeout
  2020-08-10 12:13:21,326 [INFO] [network] [read loop] do read err: read udp 127.0.0.1:55373->127.0.0.1:5300: i/o timeout
  2020-08-10 12:13:22,328 [INFO] [network] [read loop] do read err: read udp 127.0.0.1:55373->127.0.0.1:5300: i/o timeout
  ```
  
