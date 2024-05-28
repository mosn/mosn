## 使用慢启动来控制集群流量分布

## 简介

+ 该样例工程演示了如何配置集群流量分布的慢启动规则

## 准备

需要一个编译好的MOSN程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/slow-start-sample/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```

## 目录结构

```
main        # compiled MOSN
config.json # configure of MOSN
server.go   # mocked http server
```

## 运行说明

### 启动多个HTTP server

启动多个HTTP server，并指定每个HTTP server的端口（端口号`8080`、`8081`、`8082`已经配置在了`config.json`）

```
go run server.go -port=${port}
```

### 启动MOSN

+ 使用配置来运行MOSN

```
./main start -c config.json
```

### 使用CURL进行验证

最开始，所有的服务器都在相同的时间启动，此时慢启动没有发生作用

```shell
$ for i in $(seq 1 12); do curl localhost:2046; echo; done
8080
8081
8082
8081
8082
8082
8080
8081
8082
8081
8082
8082
```

杀掉端口号为`8082`的服务器，并在一个健康检查周期后启动。在服务器启动后的第一个健康检查周期后，再次执行相同的命令，可以发现端口号为`8082`的服务器的权重在逐渐增加

```shell
$ for i in $(seq 1 12); do curl localhost:2046; echo; done
8082
8081
8080
8081
8082
8081
8080
8081
8082
8081
8080
8081
```