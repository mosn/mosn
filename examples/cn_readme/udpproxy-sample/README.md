## 使用 MOSN 作为UDP 代理

## 简介

+ 该样例工程演示了如何配置使得MOSN作UDP Proxy代理
+ MOSN收到一个UDP请求，会根据请求得源地址、目的地址(不配置则为任意地址)转发到对应的cluster

## 准备

需要一个编译好的MOSN程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/udpproxy-sample/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 目录结构

```
main          // 编译完成的MOSN程序
udp.go        // 模拟的UDP Server
config.json   // 非TLS的配置
```

## 运行说明

+ 启动MOSN


```
./main start -c config.json
```

+ 转发UDP

  + 启动一个UDP Server

  ```
  go run udp.go 
  ```

  + 使用nc进行验证

  ```
  echo "hello world" | nc -4u localhost 5301
  ```
  
+ 输出

  客户端将会收到发送的相同字符
  
  ```
  ~ tiny$ echo "hello world" | nc -4u localhost 5301
  hello world
  ```
  
  UDP server端将会收到客户端发送的数据
  
  ```
  udpproxy-sample tiny$ go run udp.go
  Listening on udp port 5300 ...
  Receive from 127.0.0.1:55373, len:12, data:hello world
  ```

+ Mosn debug日志

  Mosn 将会在收到一个新的客户端地址发送的数据，并且找不到关联的会话时创建一条UDP会话，
  在5次空闲检查没有数据传输之后关闭这个UDP会话，因此我们在debug日志中会看到读超时相关的日志，
  这是符合预期的。
  
  ```
  2020-08-10 12:13:18,315 [INFO] [network] [read loop] do read err: read udp 127.0.0.1:55373->127.0.0.1:5300: i/o timeout
  2020-08-10 12:13:19,320 [INFO] [network] [read loop] do read err: read udp 127.0.0.1:55373->127.0.0.1:5300: i/o timeout
  2020-08-10 12:13:20,325 [INFO] [network] [read loop] do read err: read udp 127.0.0.1:55373->127.0.0.1:5300: i/o timeout
  2020-08-10 12:13:21,326 [INFO] [network] [read loop] do read err: read udp 127.0.0.1:55373->127.0.0.1:5300: i/o timeout
  2020-08-10 12:13:22,328 [INFO] [network] [read loop] do read err: read udp 127.0.0.1:55373->127.0.0.1:5300: i/o timeout
  ```

