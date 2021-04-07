## 使用 MOSN 作为 x_example 代理，将拓展协议打成plugin插件，在mosn中注册该插件

## 测试验证MOSN的网络代理和插件拓展

## 简介

+ 该样例工程演示了如何使用协议拓展样例，打成plugin插件，通过配置插件路径与注册方法名称等，验证MOSN作为拓展协议代理
+ 为了演示方便，MOSN监听两个端口,一个转发Client的请求，一个收到请求以后转发给Server

## 准备

+ 需要一个编译好的MOSN程序

```
cd ${projectpath}/cmd/mosn/main
go build
```

+ 示例代码目录

```
${targetpath} = ${projectpath}/examples/codes/xprotocol_with_goplugin_example/
```

+ 将编译好的MOSN程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```

+ 将协议拓展打成plugin插件，让供MOSN插拔使用

```
cd ${projectpath}/examples/codes/xprotocol_with_goplugin_example/
./make_codec.sh
```

+ 执行完上述打插件步骤后，可以找到生成的插件源文件。现在开始在MOSN配置文件client_config，server_config（client mosn和server mosn各一个）内， 
  配置plugin插件路径与方法名称等。（json并不支持//注释，使用时候请移除）

```
"third_part_codec": {
  "codecs": [
    {
    "enable": true,    //enable表示是否开启
    "type": "go-plugin",    //type表示类型为go-plugin插件
    "path": "codec.so",   //path表示插件存放相对路径
    "loader_func_name": "LoadCodec"   //loader_func_name表示所执行的方法名称，默认即LoadCodec
    }
  ]
}
```

## 运行说明

### 启动server

```
${projectpath}/examples/codes/xprotocol_with_goplugin_example/server/
go run server.go
```

### 启动MOSN

+ 使用普通配置运行非TLS加密的MOSN

启动 client 端:

```
./main start -c client_config.json
```

启动 server 端:

```
./main start -c server_config.json
```

### 使用Client进行访问

```
${projectpath}/examples/codes/xprotocol_with_goplugin_example/client/
go run client.go
```

+ 这时我们可以看到经过MOSN网络代理后返回的response

```
Hello, I am server
```

+ mson server 端与 client 端输出go-plugin codec的日志：

```
[in decodeRequest]
[out decodeRequest] payload: Hello World
[in encodeRequest] request: Hello World
[out encodeRequest]
[in decodeResponse]
[out decodeRequest] payload: Hello, I am server
[in encodeResponse] response: Hello, I am server
[out encodeResponse]
```

由上可见，mosn成功使用加载的codec.so接管数据流量。

## 如何使用go-plugin机制拓展协议

### 第一步：实现xprotocol接口
// todo

### 第二步：实现加载编解码器的方法
// todo

### 第三步：编译go-plugin
// todo

### 第四步：使用mosn加载运行go-plugin
// todo