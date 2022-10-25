## 使用 MOSN 作为TProxy 代理

## 简介

+ 该样例工程演示了如何配置使得MOSN作egress模式Transparent Proxy代理
+ 配置iptables使MOSN代理所有向外发送的请求
+ 首先选择MOSN监听的其余listener，匹配规则：优先ip、prot都命中，其次port命中
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


### 从本机启访问外部主机

```
curl http://192.168.1.5:80
this is general_server

curl http://192.168.1.5:12345
this is tproxy_server
```

