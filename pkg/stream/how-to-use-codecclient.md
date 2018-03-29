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
requestEncoder.EncodeHeaders(requestHeaders, endStream)
```

+ Encode data
```
requestEncoder.EncodeData(requestDataBuf, endStream)
```

+ Encode trailers
```
requestEncoder.EncodeTrailers(requestHeaders)
```

+ 'endStream' means further encode is not needed. for example, if we call EncodeHeaders(requestHeaders, true), stream will handle headers and send request, data and trailers will be ignored.

#### Step 4. Get response

+ 'responseDecoder' should implement 'types.StreamDecoder' interface
+ When stream gets data and decode it successfully, 'types.StreamDecoder''s interface method will be called. Example:
```
func (r *responseDecoder) OnDecodeHeaders(headers map[string]string, endStream bool) {
	// your logic
}

func (r *responseDecoder) OnDecodeData(data types.IoBuffer, endStream bool) {
    // your logic
}

func (r *responseDecoder) OnDecodeTrailers(trailers map[string]string) {
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
