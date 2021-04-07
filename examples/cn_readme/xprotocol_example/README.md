## 使用 MOSN 作为 x_example 代理，将拓展协议打成plugin插件，在mosn中注册该插件

## 测试验证MOSN的网络代理和插件拓展。

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
${targetpath} = ${projectpath}/examples/codes/xprotocol_example/
```

+ 将编译好的MOSN程序移动到示例代码目录

```
mv main ${targetpath}/
cd ${targetpath}
```

+ 将协议拓展打成plugin插件，让供MOSN插拔使用

```
cd ${projectpath}/examples/codes/xprotocol_example/x_example/regist
go build --buildmode=plugin  -o register.so register.go  //register.so为打好的插件名称，以debug启动MOSN的话，请添加 -gcflags="all=-N -l"参数
mv register.so ${projectpath}/examples/codes/xprotocol_example/   //将插件放到xprotocol_example文件夹内
```

+ 执行完上述打插件步骤后，可以找到生成的插件源文件。现在开始在MOSN配置文件client_config，server_config（client mosn和server mosn各一个）内， 
  配置plugin插件路径与方法名称等。（json并不支持//注释，使用时候请移除）

```
"third_part_codec": {
  "codecs": [
    {
    "enable": true,    //enable表示是否开启
    "type": "go-plugin",    //type表示类型为go-plugin插件
    "path": "register.so",   //path表示插件存放相对路径
    "loader_func_name": "LoadCodec"   //loader_func_name表示所执行的方法名称，默认即LoadCodec
    }
  ]
}
```

## 目录结构

```
main     // 编译完成的MOSN程序
server
    x_example_server.go   // 模拟的x_example Server
client
    x_example_client.go   // 模拟的x_example client
client_config.json // 非TLS的配置 且配置插件
server_config.json // 非TLS的配置 且配置插件
x_example  //协议拓展文件夹
register.so   打好的插件存放位置
    ......  
```

## 运行说明

### 启动x_example_server

```
${projectpath}/examples/codes/xprotocol_example/x_example/server/
go run x_example_server.go
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
${projectpath}/examples/codes/xprotocol_example/x_example/client/
go run x_example_client.go
```

+ 这时我们可以看到经过MOSN网络代理后返回的response

```
world hello ------resp

```

+ mson server 端与 client 端输出请求与响应的内容，证明请求经过网络代理，然后发出

```
[mosn decodeRequest] request 请求内容：Hello World,请求ID：1
[mosn decodeResponse] Response 响应内容：world hello,响应ID：1
```

## MOSN是如何使用plugin的，如何拓展协议

```

func readProtocolPlugin(path, loadFuncName string) error {
	p, err := goplugin.Open(path)
	if err != nil {
		return err
	}

	if loadFuncName == "" {
		loadFuncName = DefaultLoaderFunctionName
	}

	sym, err := p.Lookup(loadFuncName)
	if err != nil {
		return err
	}

	loadFunc := sym.(func() api.XProtocolCodec)
	codec := loadFunc()

	protocolName := codec.ProtocolName()
	log.StartLogger.Infof("[mosn] [init codec] loading protocol [%v] from third part codec", protocolName)

	if err := xprotocol.RegisterProtocol(protocolName, codec.XProtocol()); err != nil {
		return err
	}
	if err := xprotocol.RegisterMapping(protocolName, codec.HTTPMapping()); err != nil {
		return err
	}
	if err := xprotocol.RegisterMatcher(protocolName, codec.ProtocolMatch()); err != nil {
		return err
	}

	return nil
}
```

+ MOSN启动的时候，starter会读取配置文件信息，如果配置了plugin的话，执行readProtocolPlugin方法，获取到插件， 根据loadFuncName找到对应方法并执行，返回协议实现的实例并注册相应接口实现进去。

+ 一个完整的协议需要实现以下接口

```
type XProtocolCodec interface {
    ProtocolName() ProtocolName  //返回协议名称
    XProtocol() XProtocol       //协议相关接口
    ProtocolMatch() ProtocolMatch  //recognize if the given data matches the protocol specification or not
    HTTPMapping() HTTPMapping   //maps the contents of protocols to HTTP standard 请求状态与http协议映射关系
}
```

+ 如下为register.go内的LoadCodec(),返回的ExampleRegister结构体实例实现了上方接口
```
  func LoadCodec() api.XProtocolCodec {
    return ExampleRegister
  }
 
```