## 使用 MOSN 作为 HTTP 代理

## 简介

+ 该样例工程演示了如何基于动态数据源配置MOSN限流能力，动态数据源指的是使用第三方动态数据中心作为规则数据源
+ 限流配置为/test接口1qps

## 准备

需要一个编译好的MOSN程序
```
make build-local
```

一个Nacos服务端
```
git clone https://github.com/nacos-group/nacos-docker.git
cd nacos-docker
docker-compose -f example/standalone-derby.yaml up
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/flowcontrol-sample/
```

+ 将编译好的程序移动到示例代码目录

```
mv build/bundles/${version}/binary/mosn ${targetpath}/main
cd ${targetpath}
```


## 目录结构

```
main        // 编译完成的MOSN程序
server.go   // 模拟的Http Server
server_config.json // 非TLS的配置
client_config.json // 非TLS的配置
tls_client_config.json    // TLS配置
tls_server_config.json    // TLS配置
datasource-nacos/server_config_nacos.json        // 使用nacos动态数据源非TLS的配置
datasource-nacos/tls_server_config_nacos.json    // 使用nacos动态数据源TLS配置
mockrequest.sh // 模拟请求脚本
```

## 运行说明

### 启动一个HTTP Server

```
go run server.go
```

### nacos上创建限流规则

控制台地址：http://127.0.0.1:8848/nacos/

nacos配置
```
namespace: public
group: SENTINEL_GROUP
dataID: sentinel-dashboard-flow-rules
```

规则数据
```
[
  {
    "resource": "/test",
    "limitApp": "",
    "grade": 1,
    "threshold": 1,
    "strategy": 0
  }
]
```

### nacos数据源配置说明
```
{
  "resource_type": "nacos",
  "config": {
    "Group": "SENTINEL_GROUP",  // group, SENTINEL_GROUP
    "DataID": "",               // dataID，use APPName+"-"+"flow-rules" when empty
    "ClientConfig":{
      "NamespaceId": ""         // namespace，default public
    },
    "ServerConfigs": [
      {
        "IpAddr": "127.0.0.1",    // nacos IP
        "Port": 8848              // nacos Port
      }
    ]
  }
}
```
更多配置信息可参考https://pkg.go.dev/github.com/nacos-group/nacos-sdk-go/vo#NacosClientParam

### 启动MOSN

+ 使用配置运行非TLS加密的MOSN

启动 server 端, 使用nacos数据源:
```
./main start -c datasource-nacos/server_config_nacos.json
```

+ 使用TLS配置开启MOSN之间的TLS加密

启动 server 端, 使用nacos数据源:
```
./main start -c datasource-nacos/tls_server_config_nacos.json
```

### 使用测试脚本进行验证

```
bash mockrequest.sh
```

期望输出：

```text
request count: 1
success
request count: 2
blocked
request count: 3
blocked
```

修改nacos限流规则并发数为2
```
[
  {
    "resource": "/test",
    "limitApp": "",
    "grade": 1,
    "threshold": 2,
    "strategy": 0
  }
]
```

```
bash mockrequest.sh
```

期望输出：

```text
request count: 1
success
request count: 2
success
request count: 3
blocked
```
