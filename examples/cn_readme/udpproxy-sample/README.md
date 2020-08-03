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

### 启动MOSN


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

