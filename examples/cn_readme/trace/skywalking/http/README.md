## 使用 SkyWalking 作为 trace 框架

## 简介

+ 该样例工程演示了如何配置使得SkyWalking作MOSN的trace框架

## 准备

需要一个编译好的MOSN程序
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
main          // 编译完成的MOSN程序
server.go     // 模拟的Http Server
config.json   // MOSN配置
```

## 运行说明

### 启动一个HTTP Server

```
go run server.go
```

### 启动MOSN

+ 使用config.json 运行MOSN

```
./main start -c config.json
```

### 使用CURL进行验证

```
curl http://127.0.0.1:2046/v1
```
