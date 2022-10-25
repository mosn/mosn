## 使用 MOSN 作为 TProxy 代理

## 简介

+ 该样例工程演示了如何配置使得 MOSN 作 egress 模式 TProxy 代理
+ 配置 iptables 使 MOSN 代理发往指定目标端口的请求（可以在 iptables_config.sh 中配置）
+ 首先选择 MOSN 监听的其余 listener，匹配规则：优先 ip、prot 都命中，其次 port 命中
+ 没有匹配的 listener 则用 TProxy 代理配置的 cluster


## 准备

需要一个编译好的 MOSN 程序
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
main               // 编译完成的 MOSN 程序
config.json        // MOSN 配置
iptables_config.sh // 设置 iptables 代理端口和 mark 标记值
setup.sh           // iptables 以及路由表配置脚本
cleanup.sh         // 清除 setup.sh 的修改
```

## 运行说明


### 配置 iptables

```
sh setup.sh
```

### 启动 MOSN

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

