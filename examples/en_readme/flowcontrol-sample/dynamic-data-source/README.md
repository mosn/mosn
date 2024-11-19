## Use MOSN as HTTP Proxy

## Introduction

+ This sample project demonstrates how to configure flow control in MOSN,dynamic resource means use third party data source such as nacos as source of flow control rules
 
## Preparation

+ A compiled MOSN is needed
```
make build-local
```

+ A nacos server is needed
```
git clone https://github.com/nacos-group/nacos-docker.git
cd nacos-docker
docker-compose -f example/standalone-derby.yaml up
```

+ examples code path

```
${targetpath} = ${projectpath}/examples/codes/flowcontrol-sample/
```

+ Move the target to example code path

```
mv build/bundles/${version}/binary/mosn ${targetpath}/main
cd ${targetpath}

```

## Catelog

```
main        // compiled MOSN
server.go   // Mocked SofaRpc Server
client.go   // Mocked SofaRpc client
client_config.json // Configure without TLS
server_config.json // Configure without TLS
tls_client_config.json    // Configure with TLS
tls_server_config.json    // Configure with TLS
datasource-nacos/server_config_nacos.json        // Use nacos as Configuration data source without TLS
datasource-nacos/tls_server_config_nacos.json    // Use nacos as Configuration data source with TLS
mockrequest.sh // The scripts which mocks multiple requests for /test
```

## Operation instructions

### Create flow control rules on nacos

dashboard url：http://127.0.0.1:8848/nacos/

nacos configuration
```
namespace: public
group: SENTINEL_GROUP
dataID: sentinel-dashboard-flow-rules
```

flow control rules
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

### nacos data source configuration description
```
{
  "resource_type": "nacos",
  "config": {
    "Group": "SENTINEL_GROUP",  // group configuation，SENTINEL_GROUP
    "DataID": "",               // dataID配置，默认为空，使用APPName+"-"+"flow-rules"
    "ClientConfig":{
      "NamespaceId": ""         // namespace配置，默认为空，public
    },
    "ServerConfigs": [
      {
        "IpAddr": "127.0.0.1",    // nacos地址
        "Port": 8848              // nacos端口
      }
    ]
  }
}
```
for more configuration information refer to https://pkg.go.dev/github.com/nacos-group/nacos-sdk-go/vo#NacosClientParam

### Start HTTP Server 

```
go run server.go
```

### Start MOSN

+ Use non-TLS configs to run MOSN without TLS.

run server side with nacos data source:
```
./main start -c datasource-nacos/server_config_nacos.json
```

+ Use TLS configs to start MOSN witht TLS.

run server side with nacos data source:
```
./main start -c datasource-nacos/tls_server_config_nacos.json
```

### Use CURL for verification

```
bash mockrequest.sh
```

The output should be:

```text
request count: 1
success
request count: 2
blocked
request count: 3
blocked
```

modify flow control rules on nacos

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

The output should be:

```text
request count: 1
success
request count: 2
success
request count: 3
blocked
```
