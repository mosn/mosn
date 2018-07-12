#### Use CodecClient as a client

#### Step 1: Init CodecClient, register callbacks if needed

```
codecClient := str.NewCodecClient(protocol.Http2, connData.Connection, connData.HostInfo)
codecClient.AddConnectionCallbacks(ac)
codecClient.SetCodecClientCallbacks(ac)
codecClient.SetCodecConnectionCallbacks(ac)
```

+ connection callbacks: called on connection level event, such as connection close
+ codecclient callbacks: called on stream reset
+ codecclient connection callbacks: called on connection get a graceful close event

####Also: If you want to use the stream to realize bidirection communication, you need use NewBiDirectCodeClient 
```
codecClient := str.NewBiDirectCodeClient(protocol.Http2, connData.Connection, connData.HostInfo, SrvStreamCB)
```
where `SrvStreamCB` realizes interface `ServerStreamConnectionEventListener`, and you need to realize related func by yourself

#### Step 2. Make a new stream

```
requestStreamEncoder := codecClient.NewStream(streamId, responseDecoder)
```

When you want to make a request, just make a new stream and get a 'requestStreamEncoder', with which you can encode your request data without handling protocol details and io things.
'responseDecoder' is the decode callbacks when response is decoded successfully in stream, then you can do your logic with decoded data.

In stream encoder/decoder interface, we split request/response as three standard parts: headers, data, trailers.

#### Step 3. Send request

+ Encode headers
```
requestStreamEncoder.AppendHeaders(requestHeaders, endStream)
```

+ Encode data
```
requestStreamEncoder.AppendData(requestDataBuf, endStream)
```

+ Encode trailers
```
requestStreamEncoder.AppendTrailers(requestHeaders)
```

+ 'endStream' means further encode is not needed. for example, if we call AppendHeaders(requestHeaders, true), stream will handle headers and send request, data and trailers will be ignored.

#### Step 4. Get response

+ 'responseDecoder' should implement 'types.StreamReceiver' interface
+ When stream gets data and decode it successfully, 'types.StreamReceiver''s interface method will be called. Example:
```
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

#### Event&Exceptions
As mentioned in 'Step 1', we can register callbacks to get notified on connection/stream event.

+ Connection event callbacks:
```
func (c *callback) OnEvent(event types.ConnectionEvent) {
    // your logic
}
```

+ Stream event callbacks:
```
func (c *callback) OnStreamReset(reason types.StreamResetReason) {
	// your logic
}
```

+ Stream connection callbacks:
```
func (c *callback) OnGoAway() {
	// your logic
}
```
