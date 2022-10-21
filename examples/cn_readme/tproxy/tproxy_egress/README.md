## 使用 MOSN 作为TProxy 代理

## 简介

+ 该样例工程演示了如何配置使得MOSN作egress模式Transparent Proxy代理
+ 配置iptables使MOSN代理所有向外发送的请求
+ 优先选择MOSN监听的其余listener
+ 没有匹配的listener则用TProxy代理配置的cluster


## 准备

需要一个编译好的MOSN程序
```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/tproxy/tproxy_egress
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
client.go     // 发送请求的Client
setup.sh      // iptables以及路由表配置脚本
cleanup.sh    // 清除setup.sh的修改
config.json   // MOSN配置
```

## 运行说明


### 配置iptables

```
sh setup.sh
```

### 启动MOSN

```
./main start -c config.json
```

### 在另一台机器启动一个HTTP Server

```
go run server.go 0.0.0.0:8080
```

### 在本机启动Client并指定发送端口和目标addr

```
go run client.go 12345 192.168.0.5:8080
```

