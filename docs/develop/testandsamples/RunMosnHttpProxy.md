## 转发标准 HTTP 协议

该样例工程演示了如何配置 SOFAMesh 来转发标准 HTTP 协议，而 SOFAMesh 之间的协议是 HTTP/2。

## 准备

需要一个编译好的 SOFAMesh 程序:
```bash
cd ${projectpath}/pkg/mosn
go build
```

将编译好的程序移动到当前目录，目录结构如下 

```bash
mosn        //Mesh程序
server.go   //HTTP Server
server.json //HTTP Server的配置
client.json //HTTP Client的配置
```

## 运行说明

### 启动一个 HTTP Server

```bash
go run server.go
```

### 启动代理 HTTP Server 的 Mesher

```bash
./mosn start -c server.json
```

### 启动代理 HTTP Client 的 Mesher

```bash
./mosn start -c client.json
```

### 使用 CURL 进行验证

+ 按照默认的配置设置，HTTP Server 监听本地 `8080` 端口，HTTP Client 代理监听本地 `2046` 端口
+ Mesher 代理配置转发请求为 Header 中包含 `service:com.alipay.test.TestService:1.0`

```bash
//直接访问 HTTP Sever，观察现象
curl http://127.0.0.1:8080
//能收到 HTTP Server 返回的结果
curl --header "service:com.alipay.test.TestService:1.0" http://127.0.0.1:2046
//不能收到 HTTP Server 返回的结果(其实是返回了 404 Not Found）
curl http://127.0.0.1:2046
```

+ 可以按照说明修改配置，进行不同的测试与验证