# 使用 Prometheus 作为 插件实现

## 简介

该样例工程演示了如何配置使得 [Prometheus](https://prometheus.io/) 作为 MOSN 的 插件实现。

需要一个编译好的 MOSN 程序。

```bash
cd ${projectpath}/cmd/mosn/main
go build
```

编译 tracer 插件
``` bash
cd ${projectpath}/examples/codes/plugin/metrics
go build -buildmode=plugin -o prometheus.so prometheus.go
```

获取示例代码目录。

```bash
${targetpath} = ${projectpath}/examples/codes/plugin/metrics
```

将编译好的程序移动到示例代码目录。

```bash
mv main ${targetpath}/
cd ${targetpath}
```

## 目录结构

下面是 SkyWalking 的目录结构。

```bash
* metrics
.
├── config.json
├── main
├── prometheus.go
└── prometheus.so
```
## 运行说明

启动 MOSN。

```bash
./main start -c config.json
```

存在如下日志证明加载成功

```
[INFO] [mosn] [init metrics] create metrics sink: prometheus
```

验证功能

```bash
curl localhost:11003/metrics
```