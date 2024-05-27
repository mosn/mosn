# 使用 SkyWalking 作为 trace 实现

## 简介

该样例工程演示了如何配置使得 [SkyWalking](http://skywalking.apache.org/) 作为 MOSN 的 trace 实现。

## 准备

安装 docker & docker-compose。

1. [安装docker](https://docs.docker.com/install/)
1. [安装docker-compose](https://docs.docker.com/compose/install/)

需要一个编译好的 MOSN 程序。

```bash
cd ${projectpath}/examples/codes/trace/skywalking-plugin/http/main
go build
```

编译 tracer 插件
``` bash
cd ${projectpath}/examples/codes/trace/skywalking-plugin/plugins
go build -buildmode=plugin -o tracer.so .
```

获取示例代码目录。

```bash
${targetpath} = ${projectpath}/examples/codes/trace/skywalking-plugin/http/
```

将编译好的程序移动到示例代码目录。

```bash
mv main ${targetpath}/
cd ${targetpath}
```

## 目录结构

下面是 SkyWalking 的目录结构。

```bash
* skywalking
└─── http
│           main                           # 编译完成的 MOSN 程序
|           server.go                      # 模拟的 Http Server
|           clint.go                       # 模拟的 Http Client
|           config.json                    # MOSN 配置
|           skywalking-docker-compose.yaml # skywalking docker-compose
|           tracer.so                      # tracer plugin 
```

## 运行说明

启动 SkyWalking oap & ui。

```bash
docker-compose -f skywalking-docker-compose.yaml up -d
```

启动一个 HTTP Server。

```bash
go run server.go
```

启动 MOSN。

```bash
./main start -c config.json
```

启动一个 HTTP Client。

```bash
go run client.go
```

打开 [http://127.0.0.1:8080](http://127.0.0.1:8080/) 查看 SkyWalking-UI。
