## 使用 MOSN 作为 Dubbo 代理
 
## 简介

+ 该样例工程演示了如何配置使得MOSN作为Dubbo代理
+ 为了演示方便，MOSN监听两个端口,一个转发Client的请求，一个收到请求以后转发给Server

## 准备

+ 需要一个编译好的MOSN程序

```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/dubbo/
```

+ 将编译好的程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```


## 目录结构

```
main        // 编译完成的MOSN程序
dubbo-examples/   // 包含java编写的dubbo provider和dubbo consumer
config.json // 非TLS的配置
run.sh     // 打包和启动dubbo provider和dubbo consumer脚本
```

## 运行说明

### 启动Dubbo RPC Provider

```
sh run.sh server
```

### 启动MOSN

+ 使用config.json 运行非TLS加密的MOSN

```
./main start -c config.json

```


### 启动Dubbo RPC consumer进行访问

```
sh run.sh client
```

