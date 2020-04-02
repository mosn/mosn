## 使用 SkyWalking 作为 trace 实现

## 简介

+ 该样例工程演示了如何配置使得[SkyWalking](http://skywalking.apache.org/)作为MOSN的trace实现

## 准备

+ 安装docker & docker-compose

1. [安装docker](https://docs.docker.com/install/) <br>
1. [安装docker-compose](https://docs.docker.com/compose/install/)

+ 需要一个编译好的MOSN程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/trace/skywalking/http/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 目录结构

```
main                           // 编译完成的MOSN程序
server.go                      // 模拟的Http Server
config.json                    // MOSN配置
skywalking-docker-compose.yaml // skywalking docker-compose
```

## 运行说明

### 启动SkyWalking oap & ui
```bash
docker-compose -f skywalking-docker-compose.yaml up -d
```

### 启动一个HTTP Server

```bash
go run server.go
```

### 启动MOSN

+ 使用config.json 运行MOSN

```bash
./main start -c config.json
```

### 启动一个Http Client
```bash
go run client.go
```

### 查看SkyWalking-UI
[http://127.0.0.1:8080](http://127.0.0.1:8080)