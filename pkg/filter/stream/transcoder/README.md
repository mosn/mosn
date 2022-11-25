##  Api 介绍

- 实现Transcoder接口

```go
type Transcoder interface {
	// Accept
	Accept(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) bool
	// TranscodingRequest
	TranscodingRequest(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) (api.HeaderMap, api.IoBuffer, api.HeaderMap, error)
	// TranscodingResponse
	TranscodingResponse(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) (api.HeaderMap, api.IoBuffer, api.HeaderMap, error)
}
```

Transcoder的接口实现如上，包含三方面内容：<br />
一、实现Accept方法，要求返回一个bool类型，表示是否进行协议转换，false表示不进行协议转换。<br />
二、实现TranscodingRequest方法，对请求报文的headers、buf、trailers做转换，返回新协议报文的headers、buf、trailers。<br />
三、实现TranscodingResponse方法，对响应报文的headers、buf、trailers做转换，返回新协议报文的headers、buf、trailers。在协议转换失败会存在两类场景，场景一 :下游不健康，导致没有收到响应;场景二 :收到响应解码转换失败。开发者都需要反馈对应协议的异常报文结构返回。如果返回的为 nil，nil，nil，err。可能会导致客户端无法收到响应。


- 实现 LoadTranscoderFactory 方法
```go
func LoadTranscoderFactory(cfg map[string]interface{}) transcoder.Transcoder {
	return &xr2sp{cfg: cfg}
}
```

