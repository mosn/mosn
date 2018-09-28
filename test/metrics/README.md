## 说明

+ 以Proxy模式（Client-MOSN-Server) 模拟一个运行环境
+ 通过"API" 发送请求、获取统计数据
+ 手动测试、验证（根据输出判断)

## 启动

+ go build 编译出bin
+ ./metircs -p=${protocol}
+ -p 指定运行的协议,支持http1,http2,sofarpc

## TEST API 

### 发送一个请求

+ curl "http://127.0.0.1:8081/send"

### 销毁一个连接

+ curl "http://127.0.0.1:8081/destroy"


### 获取统计数据

+ curl "http://127.0.0.1:8081/stats"

## 特别说明

+ HTTP/HTTP2使用net/http的Client, 连接池不受控制，策略是尽量保持长连接，destory API没有意义
+ SofaRPC以BoltV1为例, 在初次请求会建立一个连接，后面再没有destory的情况话，不会建新的连接
+ 上述的连接情况指"Client"的连接请求，请求TEST API实际上是调用"Client", stats是获得mosn的统计数据
