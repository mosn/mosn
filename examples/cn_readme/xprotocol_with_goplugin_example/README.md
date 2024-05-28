## 如何使用go-plugin机制拓展协议

### 第一步：实现XProtocolCodec接口

```
type XProtocolCodec interface {
	ProtocolName() ProtocolName

	XProtocol() XProtocol

	ProtocolMatch() ProtocolMatch

	HTTPMapping() HTTPMapping
}

```
+ XProtocolCodec接口定义如上，包含四方面内容
  
  一、实现ProtocolName方法，要求返回一个字符串类型的ProtocolName。
  
  二、实现XProtocol接口，里面定义了更详细的接口内容，比如协议编解码，心跳，是否支持pool，自动生成requestId等。
  
  三、实现ProtocolMatch方法，返回参数是一个func,根据传入的byte数组，返回协议匹配成功/失败/重试。
  
  四、实现HTTPMapping接口，里面只有一个MappingHeaderStatusCode方法，关于协议和http状态码的关系映射。


+ 实现细节可以参考xprotocol_with_goplugin_example下codec内的代码及其功能

  
    command.go 里面定义了request与response的结构体，request要求实现api.XFrame接口的内容，response要求实现api.XRespFrame接口的内容
             如果response采用了类似的结构体组合，一定要记得重写GetStreamType方法。

    decoder.go 里面decodeRequest decodeResponse 对于拦截到的流量，进行分析，是否符合报文，符合的话返回request response以方便后面使用。

    encoder.go 里面encodeRequest encodeResponse 负责最终出站的流量处理。

    mapping.go 实现HTTPMapping接口，处理协议和http状态码的映射。

    matcher.go 根据报文规则，返回成功/失败。

    protocol.go 流量进出的路口，判断属于请求还是响应，然后交给decoder或者encoder处理。

    types.go  定义了报文和协议需要的一些常量。

    api.go    Codec结构体组合了实现上述xprotocol接口的结构体，后文会说明使用目的。


### 第二步：实现加载编解码器的方法
+ mosn如何加载编解码器，请阅读conn.go里面的handleFrame和handleRequest和handleResponse代码，
  请求与响应流量的进出，最终会经过Encode Decode，然后加载各自的编解码器。
  下面讲解example示例代码如何做的：
  
1.首先要理解报文，基本都离不开报文头和报文体，所有的编解码都围绕报文操作
    如下是示例协议的报文头规则，magic固定为x，作为协议的辨识标志，type指报文的类型，心跳，还是message消息等。
    dir指报文是属于请求还是响应，
    requestId很好理解，请求与响应的唯一标识。
    payloadLength，发送的消息体的长度，非常重要的内容。
    payload bytes，存放具体消息的部分。
```

  * Request command
* 0     1     2           4           6           8          10           12          14         16
* +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
* |magic| type| dir |      requestId        |     payloadLength     |     payload bytes ...       |
* +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+
*
* Response command
* 0     1     2     3     4           6           8          10           12          14         16
* +-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+-----+
* |magic| type| dir |      requestId        |   status  |      payloadLength    | payload bytes ..|
* +-----------+-----------+-----------+-----------+-----------+-----------+-----------+-----------+  

```

2.理解了报文的结构之后，按照mosn流量进出的顺序来实现具体编解码器。byte流量进入，我们需要判断它是否符合报文规则，细节请看protocol.go的Decode方法，这里依据了三点；

      一、byte数组长度不能小于最小报文的长度
      二、byte数组的第一个字节要求符合Magic
      三、dir的类型为req或者resp

Decode方法判断它符合报文之后，交给具体的decodeRequest decodeResponse处理，然后根据报文规则把byte数组内容放入Request Response结构体对象，以便mosn后续使用。
byte流量发出，mosn需要将Request Response结构体对象再还原成byte数组，然后发出，细节请看protocol.go的Encode方法。判断它是req还是说resp，然后encoder.go的具体方法内，依据报文规则，转为byte数组然后由downstream发出。


### 第三步：编译go-plugin
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
+ 其中配置的方法LoadCodec，定义在api.go，它返回了一个实现上述XProtocolCodec接口的结构体实例。


### 第四步：使用mosn加载运行go-plugin
+ 为了演示方便，MOSN监听两个端口,一个转发Client的请求，一个收到请求以后转发给Server

#### 准备


+ 编译mosn和so
```
./make_codec.sh
```


#### 运行说明

##### 启动server

```
${projectpath}/examples/codes/xprotocol_with_goplugin_example/
go run server.go
```

##### 启动MOSN

+ 使用普通配置运行非TLS加密的MOSN

启动 client 端:

```
./main start -c client_config.json
```

启动 server 端:

```
./main start -c server_config.json
```

##### 使用Client进行访问

```
${projectpath}/examples/codes/xprotocol_with_goplugin_example/
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
