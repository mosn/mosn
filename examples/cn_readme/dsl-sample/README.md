## 使用 DSL 控制 MOSN 请求处理流程

## 简介

+ 该样例工程演示了如何配置 DSL filter 来实现请求流程的处理
+ 

## 准备

需要一个编译好的 MOSN 程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/dsl-sample/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 目录结构

```
main        // 编译完成的 MOSN 程序
config.json // MOSN 的配置
```

## 运行说明

### 启动 MOSN

+ 使用 config.json 启动 MOSN

```
./main start -c config.json
```

### 使用如下命令验证 DSL 的处理


#### 场景一

```
curl localhost:2046 -v -H "host:dslhost"
```

期望输出：

```text
* About to connect() to localhost port 2046 (#0)
*   Trying 127.0.0.1...
* Connected to localhost (127.0.0.1) port 2046 (#0)
> GET / HTTP/1.1
> User-Agent: curl/7.29.0
> Accept: */*
> host:dslhost
> 
< HTTP/1.1 200 OK
< Date: Mon, 20 Jul 2020 15:55:06 GMT
< Content-Type: text/plain; charset=utf-8
< Content-Length: 10
< Host: dslhost
< User-Agent: curl/7.29.0
< Accept: */*
< Dsl: dsl
< Upstream_addr: localhost
< 
* Connection #0 to host localhost left intact
hello DSL!
```

#### 场景二

```
curl localhost:2046 -v
```


期望输出：
```text
* About to connect() to localhost port 2046 (#0)
*   Trying 127.0.0.1...
* Connected to localhost (127.0.0.1) port 2046 (#0)
> GET / HTTP/1.1
> User-Agent: curl/7.29.0
> Host: localhost:2046
> Accept: */*
> 
< HTTP/1.1 404 Not Found
< Date: Mon, 20 Jul 2020 15:55:12 GMT
< Content-Length: 0
< Host: localhost:2046
< User-Agent: curl/7.29.0
< Accept: */*
< Dsl: dsl
< Upstream_addr: localhost
< 
* Connection #0 to host localhost left intact
```
