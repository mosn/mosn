# 概述

为 mosn 加入 opentelemetry trace 的链路追踪功能。

# 项目结构

- mosn
    - pkg
        - trace
            - otel

# 启动

二开逻辑：

1、在项目启动时，我们需要把`otel`插件注册进mosn的trace driver中。因此在`mosn/pkg/trace/otel/driver.go`中注册driver

```go
const (
    DriverName = "otel"
)

func init() {
    trace.RegisterDriver(DriverName, NewOtelImpl())
}

```

然后需要在mosn的启动文件中引入这个包`mosn/cmd/mosn/main/mosn.go`

```go
import (
    _ "mosn.io/mosn/pkg/trace/otel"
)
```

2、把关键数据 inject 入 otel 的 span 中

查看`mosn/pkg/trace/otel/span.go`文件，如果想要添加额外的数据，则可以修改下面的函数就可以

```go
// SetRequestInfo record current request info to Span
func (s *Span) SetRequestInfo(reqInfo api.RequestInfo) {

    s.otelSpan.SetAttributes(semconv.HTTPResponseStatusCode(reqInfo.ResponseCode()))

    if tcpAddr, ok := reqInfo.DownstreamLocalAddress().(*net.TCPAddr); ok {
        s.otelSpan.SetAttributes(semconv.ServerAddress(tcpAddr.IP.String()))
        s.otelSpan.SetAttributes(semconv.ServerPort(tcpAddr.Port))
    }

    if tcpAddr, ok := reqInfo.DownstreamRemoteAddress().(*net.TCPAddr); ok {
        s.otelSpan.SetAttributes(semconv.ClientAddress(tcpAddr.IP.String()))
        s.otelSpan.SetAttributes(semconv.ClientPort(tcpAddr.Port))
    }
}

func (s *Span) SetRequestHeader(header http.RequestHeader) {
    s.otelSpan.SetAttributes(semconv.NetworkProtocolName(string(header.Protocol())))
    s.otelSpan.SetAttributes(semconv.HTTPMethod(string(header.Method())))
    s.otelSpan.SetAttributes(semconv.HTTPURL(string(header.RequestURI())))
}
```

3、实际使用
然后我们在项目启动的配置文件中配置使用 otel 就可以了。例子如下：

```json
{
  "tracing": {
    "enable": true,                     // 是否启动 tracing 功能
    "driver": "otel",                   // driver 选择 otel
    "config": {
      "service_name": "sample_sidecar", // 当前sidecar上报到 service_name 的服务名称，建议【应用服务名_sidecar】,不设置则默认值为：default_sidecar
      "report_method": "console",       // 上报方式 {console|http|grpc}，不设置默认 console
      "endpoint": "127.0.0.1:6831",     // 当前 otle 的上报地址，非 console 方式不能为空
      "config_map": {                   // 额外属性会被添加到 Resource 中，值需要为 string，具体参考 https://opentelemetry.io/docs/instrumentation/go/resources/
        "attribute1": "value",
        "attribute2": "value",
        "attribute3": "value"
      }
    }
  }
}
```