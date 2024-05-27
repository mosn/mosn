## MOSN使用xds-server 实现动态配置
 
## 简介

+ 该样例工程演示了如何配置使得MOSN使用自定义xds server来实现动态配置
+ 该样例工程仅适用istio 1.10.0 及以后,envoy v3
## 准备

+ 需要一个编译好的MOSN程序

```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/xds-server-sample/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 运行说明

### 启动 upstream server

```
  go run cmd/upstream/main.go 

```

### 启动xds-server


```
go run cmd/server/main.go

```

### 启动 mosn:

```
./main start -c config.json
```


## 验证

请求 upstream 与 请求mosn返回结果一致
```bash
curl localhost:18080
curl localhost:10000
```

返回

```bash
Default message
```
 
修改confg/config.yaml,可以使mosn修改对应的行为