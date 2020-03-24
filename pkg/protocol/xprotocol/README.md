# XProtocol Development Guide

This guide is written for potential developer who wants to extend their own RPC protocols in XProtocol framework.

# Introduction

Make sure you have already known the [basic architecture of MOSN](https://mosn.io/zh/docs/concept/core-concept/).

XProtocol framework is designed for making extension of new protocols much easier. The framwork providing stream-level `common behaviour template` and protocol-level `interfaces` for typical RPC protocols, like SOFABolt and Dubbo.

All you need is to write protocol-level extension and implements the `interfaces` defined by XProtocol.

TODO: pic

# Quick-start

In the section, "x_example" protocol(code folder: pkg/protocol/xprotocol/example) will be taken as example to demonstrates how to develop a brand new protocol with XProtocol framework.

Generally speaking, There are three steps:
1. Confirm the protocol format
2. Define packet models 
3. Define protocol behaviours 

## Confirm the protocol format

First of all, take a glance at the protocol format. All of the following works are dependent with the protocol design itself.

```
/**
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
 */
```

Once the format is determined, we should be aware of the binary bytes format, the packet model layout and how to encode/decode them.

## Define packet models

Based on the protocol format, now we can define our packet models. Models will be treated as programmable objects in higher-level, like in Proxy and Router module. In particular, XProtocol framework defined a standard set of interfaces to abstract the ability XProtocol needed. Packet models should implement the interfaces by case.

### Define model structure

Usually the struct is determined by the protocol format, keep the logical mapping between them.

```go
type Request struct {
	Type       byte
	RequestId  uint32
	PayloadLen uint32
	Payload    []byte
	Content    types.IoBuffer
}

type Response struct {
	Request
	Status uint16
}
```

### Implement XProtocol interfaces

Request and response should implement the `XFrame` and `XRespFrame` interface, XProtocol framework is dependent on the abilities provided by the interfaces, such as `Multiplexing` and `HeartbeatPredicate`.

`Multiplexing` provides the ability to distinguish multi-requests in single-connection by recognize 'request-id' semantics. And `HeartbeatPredicate` provides the ability to judge if current frame is a heartbeat, which is usually used to make connection keepalive

```go
// XFrame represents the minimal programmable object of the protocol.
type XFrame interface {
	// TODO: make multiplexing optional, and maybe we can support PING-PONG protocol in this framework.
	Multiplexing

	HeartbeatPredicate

	GetStreamType() StreamType

	GetHeader() types.HeaderMap

	GetData() types.IoBuffer
}

// XRespFrame expose response status code based on the XFrame
type XRespFrame interface {
	XFrame

	GetStatusCode() uint32
}
```

## Define protocol behaviours

Beside the concrete packet frame, XProtocol framework also need some ability which should provided by the general protocol. And this is also represent in the interface form as following:

```go
// XProtocol provides extra ability(Heartbeater, Hijacker) to interacts with the proxy framework based on the Protocol interface.
// e.g. A request which cannot find route should be responded with a error response like '404 Not Found', that is what Hijacker
// interface exactly provides.
type XProtocol interface {
	types.Protocol

	Heartbeater

	Hijacker
}
```

What you need is to implement the protocol and register it into XProtocol framework.

```go
func init() {
	xprotocol.RegisterProtocol(ProtocolName, &proto{})
}

```

At last, don't forget to import your implementation at the entry point `cmd/mosn/main` and speficy the sub protocol in configuration.

```go
import _ "mosn.io/mosn/pkg/protocol/xprotocol/example"
```

```json
...
{
  "type": "proxy",
  "config": {
    "downstream_protocol": "X",
    "upstream_protocol": "X",
    "router_config_name": "router",
    "extend_config": {
      "sub_protocol": "x_example"
    }
  }
}
...
```

That's it, the 'x_example' protocol is done!

# Tips, Tricks and Hacks

## Stateless codec with MOSN goroutine model

> No blocking

The most common goroutine model for I/O multiplexing codec is like the following:

- FrameReader: responsible for continually read data from connection, and parse the raw data into frame. Frames will be pass to the `FrameProcessor` by channels.
- FrameProcessor: responsible for continually read frames from `FrameReader`, and process the frames properly.

And the codec logic is embed in the FrameReader, usually write as blocking mode(Take the benefit of golang netpoll mode). So the logic flow is like: 
1. (entry code) FrameReader: start frame decode loop.
2. FrameReader: blocking read data util it meets the minimal length(usually the length of frame header).
3. FrameReader: parse the header and blocking read data util it meets the length of a whole frame.
4. FrameReader: frame parse success, pass it to FrameProcessor by channel.

But in MOSN its kind different, you cannot directly read from the connection, but interacts with the I/O buffer which MOSN provides.
1. (entry code) ReadGoroutine: start data read loop.
2. ReadGoroutine: read as much as possible, and fill it into the I/O buffer
3. ReadGoroutine: notify the listeners and pass the I/O buffer to them.

In conclusion, your codec codes is running at the `read goroutine`, and you cannot directly read from `net.Conn`, which means no netpoll works here. So if your code has blocking part to wait for more data available, the whole `read goroutine` is blocked and no more data would fill into the I/O buffer, which causes the dead lock.

## Buffer management for I/O multiplexing

> Alloc new buffer for your data

I/O buffer is managed by the `read goroutine`, be careful about its **lifecycle**.For example, while new data is available, `read goroutine` will notify the ReadFilter listeners, including the codec impls. And decoder will receive a parameter with type `types.IoBuffer`, which contains the readable data.

After decode finished you will get a protocol-specific model, like bolt.Request. Make sure no pointers/references in the model points to the `types.IoBuffer` parameter, because it might be recycled by the `read goroutine`. 

## Hack of Append headers/data/trailers 

> Pass your model as `headers` and ignore data/trailers

The typical stream layer is designed for most common protocols, like HTTP, HTTP/2, etc. So it has the traditional parts for transmission separation: headers, data, trailers. It's good for some scenarios, like upload files by streaming mode, etc.

But for most RPC protocols, there is no need to divide the transmission of the request into several parts. And, the payload size usually is not large, which means streaming is not critical. On the contrary, use `append headers/data/trailers` for RPC protocols might break the memory recycle usage for the whole request.

So we use a trick here, decorate your model as `types.HeaderMap` and pass it by append headers. The XProtocol framework would recognize this and ignore append data/trailers.

 