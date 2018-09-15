# 使用 MOSN 的 CodecClient 作为客户端

### 步骤一: 初始化 CodecClient

同时如果有需要，请注册对应的 callback 来监听连接上的事件。

```go
codecClient := str.NewCodecClient(context.Background(), protocol.Http2, connData.Connection, connData.HostInfo)
codecClient.AddConnectionCallbacks(ac)
codecClient.SetCodecClientCallbacks(ac)
codecClient.SetCodecConnectionCallbacks(ac)
```

+ connection callbacks：监听连接层面的事件，比如”连接关闭"、"连接超时"等
+ codecclient callbacks：当 client 上发送的 stream 被 reset 的时候会调用
+ codecclient connection callbacks：当 client 上的连接优雅关闭的时候会调用

如果你希望使用双工通信，那么需要使用 NewBiDirectCodeClient 来创建 `codecClient`

```go
codecClient := str.NewBiDirectCodeClient(protocol.Http2, connData.Connection, connData.HostInfo, SrvStreamCB)
```
其中，你需要自己实现 `ServerStreamConnectionEventListener`这个接口，即`SrvStreamCB`

### 步骤二. 创建 stream

```go
requestStreamEncoder := codecClient.NewStream(streamId, responseDecoder)
```

当你想发起一个请求的时候，直接通过 `NewStream` 来创建，并返回 `requestStreamEncoder`；
通过 `requestStreamEncoderrequest`，可以直接发送请求数据，而不用关心底层的协议处理和通信细节；
入参 `responseDecoder` 是收到 resp 时候的 callback，你可以根据自己需要的处理方式来实现对应的回调接口；

在 stream 的 `encoder/decoder` 接口中，我们将 `request/response` 数据分为三个部分：headers、data、trailers。

### 步骤三. 发送 stream

+ 发送 headers

    ```go
    requestStreamEncoder.AppendHeaders(requestHeaders, endStream)
    ```

+ 发送 data

    ```go
    requestStreamEncoder.AppendData(requestDataBuf, endStream)
    ```

+ 发送 trailers

    ```go
    requestStreamEncoder.AppendTrailers(requestHeaders)
    ```

+ `endStream` 是一个标识符，用来标识后序没有数据需要处理。

   例如, 如果这里调用 AppendHeaders(requestHeaders, true)，那么 stream 层将处理 headers 并发送数据，data and trailers 将被忽略。

### 步骤四. 接收 response

+ `responseDecoder` 需要使用 `types.StreamReceiver` 接口

+ 当 stream 收到数据并 decode 成功后，`types.StreamReceiver` 接口会被回调

    示例如下：

    ```go
    func (r *responseDecoder) OnReceiveHeaders(headers map[string]string, endStream bool) {
        // your logic
    }
    
    func (r *responseDecoder) OnReceiveData(data types.IoBuffer, endStream bool) {
        // your logic
    }
    
    func (r *responseDecoder) OnReceiveTrailers(trailers map[string]string) {
        // your logic
    }
    
    func (r *responseDecoder) OnDecodeError(err error, headers map[string]string) {
       // your logic
    }
    ```

## 事件&异常
正如 步骤一中提到的，可以注册相应的 callback 来获取 connection/stream 相关的事件。

+ Connection event callbacks：

    ```go
    func (c *callback) OnEvent(event types.ConnectionEvent) {
        // your logic
    }
    ```

+ Stream event callbacks：

    ```go
    func (c *callback) OnStreamReset(reason types.StreamResetReason) {
        // your logic
    }
    ```

+ Stream connection callbacks：

    ```go
    func (c *callback) OnGoAway() {
        // your logic
    }
    ```