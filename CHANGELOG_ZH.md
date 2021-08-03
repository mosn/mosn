# 更新日志

## v0.23.0

### 新功能

- 新增 networkfilter:grpc，支持通过 networkfilter 扩展方式在 MOSN 中实现可复用 MOSN 其他能力的 grpc server [@nejisama](https://github.com/nejisama) [@zhenjunMa](https://github.com/zhenjunMa)
- StreamFilterChain 新增遍历调用的扩展接口 [@wangfakang](https://github.com/wangfakang)
- bolt 协议新增 HTTP 403 状态码的映射 [@pxzero](https://github.com/pxzero)
- 新增主动关闭 upstream 连接的能力 [@nejisama](https://github.com/nejisama)

### 优化

- networkfilter 配置解析能力优化 [@nejisama](https://github.com/nejisama)
- proxy 配置解析支持按照协议扩展，配置解析时机优化 [@nejisama](https://github.com/nejisama)
- TLS 连接新增证书缓存，减少重复证书的内存占用 [@nejisama](https://github.com/nejisama)
- 优化 Quick Start Sample [@nobodyiam](https://github.com/nobodyiam)
- 优化默认路由处理时的 context 对象生成 [@alpha-baby](https://github.com/alpha-baby)
- 优化 Subset LoadBalancer 的创建函数接口 [@alpha-baby](https://github.com/alpha-baby)
- 新增使用 so plugin 扩展方式接入协议扩展的示例 [@yichouchou](https://github.com/yichouchou)
- 优化 makefile 中获取 GOPATH 环境变量的方式 [@bincherry](https://github.com/bincherry)
- 支持 darwin + arrch64 架构的编译 [@nejisama](https://github.com/nejisama)
- 优化日志打开方式 [@taoyuanyuan](https://github.com/taoyuanyuan)

### Bug 修复

- HTTP1 修复 URL 处理编码问题 [@morefreeze](https://github.com/morefreeze)
- HTTP1 修复 URL 处理大小写敏感错误问题 [@GLYASAI](https://github.com/GLYASAI)
- TLS 修复 SM4 套件异常处理时存在的内存泄漏问题 [@william-zk](https://github.com/william-zk)


## v0.22.0

### 新功能

- 新增 Wasm 扩展框架 [@antJack](https://github.com/antJack)
- XProtocol 协议新增 x-bolt 子协议，支持基于 Wasm 的协议编解码能力 [@zonghaishang](https://github.com/zonghaishang)
- 支持自动协议识别失败时根据 SO_ORIGINAL_DST 进行自动转发报文的能力 [@antJack](https://github.com/antJack)
- XProtocol 支持 Go Plugin 模式扩展 [@fdingiit](https://github.com/fdingiit)
- 新增网络扩展层 [@wangfakang](https://github.com/wangfakang)
- 支持 Istio xDS v3 API [@champly](https://github.com/champly) 所属分支：[istio-1.7.7](https://github.com/mosn/mosn/tree/istio-1.7.7)

### 优化

- 去除 StreamFilter 配置解析中多余的路径清洗 [@eliasyaoyc](https://github.com/eliasyaoyc)
- 支持为 StreamFilterChain 设置统一的回调接口 [@antJack](https://github.com/antJack)
- FeatureGate 支持不同启动阶段执行，去除 FeatureGate 状态判断的全局锁 [@nejisama](https://github.com/nejisama)
- Http2 模块新增对 trace 能力的支持 [@OrezzerO](https://github.com/OrezzerO)


### 重构

- 新增 StageManager，将 MOSN 启动流程划分为四个可自定义的阶段 [@nejisama](https://github.com/nejisama)
- 统一 XProtocol 模块的类型定义，移动至 mosn.io/api 包 [@fdingiit](https://github.com/fdingiit)
- XProtocol 接口新增 GetTimeout 方法，取代原有的变量获取方式 [@nejisama](https://github.com/nejisama)


### Bug 修复

- 修复 Proxy 中请求信息的并发冲突问题 [@nejisama](https://github.com/nejisama)
- 修复 URL 处理时的安全漏洞 [@antJack](https://github.com/antJack)
- 修复配置持久化时 Router 配置的并发冲突问题 [@nejisama](https://github.com/nejisama)


## v0.21.0

### 优化

- 升级 sentinel 版本到 v1.0.2 [@ansiz](https://github.com/ansiz)
- 读超时收缩 tls 的 read buffer，降低 tls 内存消耗 [@cch123](https://github.com/cch123)
- 增加注释，简化 xprotocol 协议连接池实现 [@cch123](https://github.com/cch123)
- 更新 mosn registry 版本 [@cadeeper](https://github.com/cadeeper) [@cch123](https://github.com/cch123)

### 重构

- 优化路由 Header 匹配逻辑，支持通用的 RPC 路由匹配 [@nejisama](https://github.com/nejisama)
- 删除原有部分常量，新增用于描述变量机制的常量 [@nejisama](https://github.com/nejisama)
- 限流模块重构，支持自定义回调扩展，可实现自定义的过滤条件，上下文信息修改等能力 [@ansiz](https://github.com/ansiz)

### Bug 修复

- 修复请求异常时 metrics 统计错误 [@cch123](https://github.com/cch123)
- 修复 http 场景转发前没有对 url 进行转义的问题 [@antJack](https://github.com/antJack)
- 修复 HTTP 协议中变量注入错误的问题，修复 HTTP2 协议中不支持路由 Rewrite 的 bug [@nejisama](https://github.com/nejisama)

### 新功能

- 支持 Domain-Specific Language 路由实现 [@CodingSinger](https://github.com/CodingSinger)
- StreamFilter 支持 go 编写的动态链接库加载的方式 [@CodingSinger](https://github.com/CodingSinger)
- 路由配置中 VirtualHost 支持 per_filter_config 配置 [@machine3](https://github.com/machine3)
- 支持 dubbo thrift 协议 [@cadeeper](https://github.com/cadeeper)

## v0.20.0

### 优化

- 优化 TCP 地址解析失败默认解析 UDS 地址的问题，地址解析前添加前缀判断 [@wangfakang](https://github.com/wangfakang)
- 优化连接池获取的尝试间隔 [@nejisama](https://github.com/nejisama)
- 支持通过全局配置关闭循环写模式 [@nejisama](https://github.com/nejisama)
- 优化协议自动识别的配置示例和测试用例 [@taoyuanyuan](https://github.com/taoyuanyuan)
- 用更高效的变量机制替换请求头 [@CodingSinger](https://github.com/CodingSinger)
- 将 WriteBufferChan 的定时器池化以降低负载 [@cch123](https://github.com/cch123)
- TraceLog 中新增 MOSN 处理失败的信息 [@nejisama](https://github.com/nejisama)
- HTTP 协议处理中，新增读完成 channel [@alpha-baby](https://github.com/alpha-baby)
- 日志轮转功能加强 [@nejisama](https://github.com/nejisama)

### 重构

- 使用的 Go 版本升级到 1.14.13 [@nejisama](https://github.com/nejisama)
- 将路由链扩展方式修改为路由 Handler 扩展方式，支持配置不同的路由 Handler [@nejisama](https://github.com/nejisama)
- MOSN 扩展配置修改，支持按照配置顺序进行解析 [@nejisama](https://github.com/nejisama)

### Bug 修复

- 修复 doubbo 版本升级至 2.7.3 之后 Provider 不可用的问题 [@cadeeper](https://github.com/cadeeper)
- 修复 netpoll 模式下，错误将 UDS 连接处理成 TCP 连接的问题 [@wangfakang](https://github.com/wangfakang)
- 修复 HTTP Header 被设置为空字符串时无法正确 Get 的问题 [@ianwoolf](https://github.com/ianwoolf)

### 新功能

- 支持新旧 MOSN 之间通过 UDS 转移配置，解决 MOSN 使用 XDS 获取配置无法平滑升级的问题 [@alpha-baby](https://github.com/alpha-baby)
- 协议自动识别支持 XProtocol [@cadeeper](https://github.com/cadeeper)
- 支持配置 XProtocol 的 keepalive 参数 [@cch123](https://github.com/cch123)
- 支持更详细的用时追踪 [@nejisama](https://github.com/nejisama)
- 支持度量指标懒加载的方式，以解决服务数目过多 metrics 空间占用过大的问题 [@champly](https://github.com/champly)
- 添加设置 XProtocol 连接池大小默认值的函数 [@cch123](https://github.com/cch123)
- 支持 netpoll 模式 [@cch123](https://github.com/cch123)
- 支持广播功能 [@dengqian](https://github.com/dengqian)
- 支持从 LDS 响应中获取 tls 配置 [@wZH-CN](https://github.com/wZH-CN)
- SDS 新增 ACK response [@wZH-CN](https://github.com/wZH-CN)

## v0.19.0

### 优化

- 使用最新的 TLS 内存优化方案 [@cch123](https://github.com/cch123)
- proxy log 优化，减少内存逃逸 [@taoyuanyuan](https://github.com/taoyuanyuan)
- 增加最大连接数限制 [@champly](https://github.com/champly)
- AccessLog 获取变量失败时，使用”-”代替 [@champly](https://github.com/champly)
- MaxProcs 支持配置基于 CPU 使用限制自动识别 [@champly](https://github.com/champly)
- 支持指定 Istio cluster 的网络 [@champly](https://github.com/champly)

### 重构

- 重构了 StreamFilter 框架，减少 streamfilter 框架与 proxy 的耦合，支持其他 network filter 可复用 stream filter 框架 [@antJack](https://github.com/antJack)

### Bug 修复

- 修复 HTTP Trace 获取 URL 错误 [@wzshiming](https://github.com/wzshiming)
- 修复 xds 配置解析时没有解析连接超时的错误 [@dengqian](https://github.com/dengqian)
- 修复变量获取 Hostname 的错误 [@dengqian](https://github.com/dengqian)
- 修复 tcp proxy 没有正确关闭连接的错误 [@dengqian](https://github.com/dengqian)
- 修复 mixer filter 缺少默认配置，导致空指针问题 [@glyasai](https://github.com/glyasai)
- 修复 HTTP2 直接响应没有正确地设置 `Content-length` 的问题 [@wangfakang](https://github.com/wangfakang)
- 修复 getAPISourceEndpoint 方法空指针问题 [@dylandee](https://github.com/dylandee)
- 修复 Write 堆积时，过多的 Timer 申请导致内存上涨的问题 [@champly](https://github.com/champly)
- 修复 Dubbo Filter 收到非法响应时，stats 统计缺失的问题 [@champly](https://github.com/champly)

## v0.18.0

### 新功能

- 新增 MOSN 配置文件扩展机制 [@nejisama](https://github.com/nejisama)
- 新增 MOSN 配置工具，提升用户配置体验 [mosn/configure](https://github.com/mosn/configure) [@cch123](https://github.com/cch123)

### 优化

- HTTP 协议 stream 处理过程中，避免多次拷贝 HTTP body [@wangfakang](https://github.com/wangfakang)
- 升级了 `github.com/TarsCloud/TarsGo` 包到 v1.1.4 版本 [@champly](https://github.com/champly)
- 补充了连接池的单元测试 [@cch123](https://github.com/cch123)
- 使用内存池减少了 TLS 连接的内存占用 [@cch123](https://github.com/cch123)
- 减少 xprotocol stream 处理过程的临界区大小，提升性能 [@cch123](https://github.com/cch123)
- 删除 `network.NewClientConnection` 方法冗余参数，删除 `streamConn` 结构体 `Dispatch` 方法 `ALPN` 检查 [@nejisama](https://github.com/nejisama)
- `StreamReceiverFilterHandler` 增加 `TerminateStream` API，可在处理流的时候传入 HTTP code 异步关闭流 [@nejisama](https://github.com/nejisama)
- client 端 TLS handshake 失败时增加降级逻辑 [@nejisama](https://github.com/nejisama)
- 修改 TLS hashvalue 计算方式 [@nejisama](https://github.com/nejisama)
- 修正 disable_log admin api typo [@nejisama](https://github.com/nejisama)

### Bug 修复

- 修复执行 `go mod tidy` 失败 [@champly](https://github.com/champly)
- 修复 MOSN 接收 XDS 消息大于 4M 时的 `ResourceExhausted: grpc: received message larger than max` 错误 [@champly](https://github.com/champly)
- 修复容错单元测试用例 [@wangfakang](https://github.com/wangfakang)
- 修复 `MOSNConfig.servers[].listeners[].bind_port` 设置为 `false` 时热重启出错 [@alpha-baby](https://github.com/alpha-baby)
- 本地写 buffer 增加超时时间，避免本地写失败导致 goroutine 过多 OOM [@cch123](https://github.com/cch123)
- 修复 TLS 超时导致死循环 [@nejisama](https://github.com/nejisama)
- 修复 `dubbo.Frame` struct 使用 `SetData` 方法之后数据没有被修改的问题 [@lxd5866](https://github.com/lxd5866)

## v0.17.0

### 新功能

- 新增最大 Header 大小限制的配置选项 [@wangfakang](https://github.com/wangfakang)
- 支持协议实现时选择是否需要 workerpool 模式，在 workerpool 模式下，支持可配置的连接并发度
  [@cch123](https://github.com/cch123)
- Listener 配置新增对 UDS 的支持 [@CodingSinger](https://github.com/CodingSinger)
- 添加在 Dubbo 协议下通过 xDS HTTP 配置进行转换的过滤器 [@champly](https://github.com/champly)

### 优化

- 优化 http 场景下的 buffer 申请 [@wangfakang](https://github.com/wangfakang)
- 优化 SDS Client 使用读写锁获取 [@chainhelen](https://github.com/chainhelen)
- 更新 hessian2 v1.7.0 库 [@cch123](https://github.com/cch123)
- 修改 NewStream 接口，从回调模式调整为同步调用的模式 [@cch123](https://github.com/cch123)
- 重构 XProtocol 连接池，支持 pingpong 模式、多路复用模式与连接绑定模式 [@cch123](https://github.com/cch123)
- 优化 XProtocol 多路复用模式，支持单机 Host 连接数可配置，默认是 1 [@cch123](https://github.com/cch123)
- 优化正则路由配置项，避免 dump 过多无用配置 [@wangfakang](https://github.com/wangfakang)

### Bug 修复

- 修复 README 蚂蚁 logo 地址失效的问题 [@wangfakang](https://github.com/wangfakang)
- 修复当请求 header 太长覆盖请求内容的问题 [@cch123](https://github.com/cch123)
- 修复 Dubbo 协议解析 attachment 异常的问题 [@champly](https://github.com/champly)

## v0.16.0

### 优化

- Logger Roller 支持自定义 Roller 的实现 [@wenxuwan](https://github.com/wenxuwan)
- StreamFilter 新增接口 SendHijackReplyWithBody [@wenxuwan](https://github.com/wenxuwan)
- 配置项新增关闭热升级选项，关闭热升级以后一个机器上可以同时存在多个不同的 MOSN 进程 [@cch123](https://github.com/cch123)
- 优化 MOSN 集成测试框架，补充单元测试 [@nejisama](https://github.com/nejisama) [@wangfakang](https://github.com/wangfakang) [@taoyuanyuan](https://github.com/taoyuanyuan)
- xDS 配置解析支持 DirectResponse 的路由配置 [@wangfakang](https://github.com/wangfakang)
- ClusterManager 配置新增 TLSContext [@nejisama](https://github.com/nejisama)

### Bug 修复

- 修复在热升级时 UDP 连接超时会导致死循环的 BUG [@dengqian](https://github.com/dengqian)
- 修复在 SendFilter 中执行 DirectResponse 会触发死循环的 BUG [@taoyuanyuan](https://github.com/taoyuanyuan)
- 修复 HTTP2 的 Stream 计数并发统计冲突的 BUG [@wenxuwan](https://github.com/wenxuwan)
- 修复 UDP 连接因读超时导致的数据丢失问题 [@dengqian](https://github.com/dengqian)
- 修复触发重试时因为协议标识丢失导致无法正确记录响应 StatusCode 的 BUG [@dengqian](https://github.com/dengqian)
- 修复 BoltV2 协议解析错误的 BUG [@nejisama](https://github.com/nejisama)
- 修复 Listener Panic 后无法自动 Restart 的 BUG [@alpha-baby](https://github.com/alpha-baby)
- 修复变量机制中 NoCache 标签无效的 BUG [@wangfakang](https://github.com/wangfakang)
- 修复 SDS 重连时可能存在并发冲突的 BUG [@nejisama](https://github.com/nejisama)

## v0.15.0

### 新功能

- 路由 Path Rewrite 支持按照正则表达式的方式配置 Rewrite 的内容 [@liangyuanpeng](https://github.com/liangyuanpeng)
- 配置新增字段： 扩展配置字段，可通过扩展配置字段自定义启动配置；Dubbo 服务发现配置通过扩展的配置字段实现 [@cch123](https://github.com/cch123)
- 支持 DSL 新特性，可以方便的对请求的处理行为进行控制 [@wangfakang](https://github.com/wangfakang)
- StreamFilter 新增流量镜像功能的扩展实现 [@champly](https://github.com/champly)
- Listener 配置新增对 UDP 的支持 [@dengqian](https://github.com/dengqian)
- 配置格式支持 Yaml 格式解析 [@GLYASAI](https://github.com/GLYASAI)
- 路由支持 HTTP 重定向配置 [@knight42](https://github.com/knight42)

### 优化

- 支持 istio 的 stats filter，可以根据匹配条件进行 metrics 的个性化记录 [@wzshiming](https://github.com/wzshiming)
- Metrics 配置支持配置 Histogram 的输出百分比 [@champly](https://github.com/champly)
- StreamFilter 新增状态用于直接中止请求，并且不响应客户端 [@taoyuanyuan](https://github.com/taoyuanyuan)
- XProtocol Hijack 响应支持携带 Body [@champly](https://github.com/champly)
- Skywalking 升级到 0.5.0 版本 [arugal](https://github.com/arugal)
- Upstream 连接 TLS 状态判断修改，支持通过 TLS 配置的 Hash 判断是否需要重新建立连接 [@nejisama](https://github.com/nejisama)
- 优化 DNS cache 逻辑，防止在 DNS 失效时可能引起的 DNS flood 问题 [@wangfakang](https://github.com/wangfakang)

### Bug 修复

- 修复开启 TLS 加密场景下，XProtocol 协议在有多个协议的场景下判断协议错误的 BUG [@nejisama](https://github.com/nejisama)
- 修复 AccessLog 中前缀匹配类型的变量不生效的 BUG [@dengqian](https://github.com/dengqian)
- 修复 Listener 配置解析处理不正确的 BUG [@nejisama](https://github.com/nejisama)
- 修复 Router/Cluster 在文件持久化配置类型中，Name 字段包含路径分隔符时会保存失败的 BUG [@nejisama](https://github.com/nejisama)

## v0.14.0

### 新功能

- 支持 Istio 1.5.X [@wangfakang](https://github.com/wangfakang) [@trainyao](https://github.com/trainyao) [@champly](https://github.com/champly)
  - go-control-plane 升级到 0.9.4 版本
  - xDS 支持 ACK，新增 xDS 的 Metrics
  - 支持 Istio sourceLabels 过滤功能
  - 支持 pilot-agent 的探测接口
  - 支持更多的启动参数，适配 Istio agent 启动场景
  - gzip、strict-dns、original-dst 支持 xDS 更新
  - 移除 Xproxy 逻辑
- Maglev 负载均衡算法支持 [@trainyao](https://github.com/trainyao)
- 新增连接池实现，用于支持消息类请求 [@cch123](https://github.com/cch123)
- 新增 TLS 连接切换的 Metrics [@nejisama](https://github.com/nejisama)
- 新增 HTTP StatusCode 的 Metrics [@dengqian](https://github.com/dengqian)
- 新增 Metrics Admin API 输出 [@dengqian](https://github.com/dengqian)
- proxy 新增查询当前请求数的接口 [@zonghaishang](https://github.com/zonghaishang)
- 支持 HostRewrite Header [@liangyuanpeng](https://github.com/liangyuanpeng)

### 优化

- 升级 tars 依赖，修复在高版本 Golang 下的编译问题 [@wangfakang](https://github.com/wangfakang)
- xDS 配置解析升级适配 Istio 1.5.x [@wangfakang](https://github.com/wangfakang)
- 优化 proxy 的日志输出 [@wenxuwan](https://github.com/wenxuwan)
- DNS Cache 默认时间修改为 15s [@wangfakang](https://github.com/wangfakang)
- HTTP 参数路由匹配优化 [@wangfakang](https://github.com/wangfakang)
- 升级 fasthttp 库 [@wangfakang](https://github.com/wangfakang)
- 优化 Dubbo 请求转发编码 [@zonghaishang](https://github.com/zonghaishang)
- 支持 HTTP 的请求最大 body 可配置 [@wangfakang](https://github.com/wangfakang)

### Bug 修复

- 修复 Dubbo Decode 无法解析 attachment 的 bug [@champly](https://github.com/champly)
- 修复 HTTP2 连接建立之前就可能创建 stream 的 bug [@dunjut](https://github.com/dunjut)
- 修复处理 HTTP2 处理 Trailer 空指针异常 [@taoyuanyuan](https://github.com/taoyuanyuan)
- 修复 HTTP 请求头默认不标准化处理的 bug [@nejisama](https://github.com/nejisama)
- 修复 HTTP 请求处理时连接断开导致的 panic 异常 [@wangfakang](https://github.com/wangfakang)
- 修复 dubbo registry 的读写锁拷贝问题 [@champly](https://github.com/champly)

## v0.13.0

### 新功能

- 支持 Strict DNS Cluster [@dengqian](https://github.com/dengqian)
- 支持 GZip 处理的 Stream Filter [@wangfakang](https://github.com/wangfakang)
- Dubbo 服务发现完成 Beta 版本 [@cch123](https://github.com/cch123)
- 支持单机故障隔离的 Stream Filter [@NeGnail](https://github.com/NeGnail)
- 集成 Sentinel 限流能力 [@ansiz](https://github.com/ansiz)

### 优化

- 优化 EDF LB 的实现，使用 EDF 重新实现 WRR LB [@CodingSinger](https://github.com/CodingSinger)
- 配置获取 ADMIN API 优化，新增 Features 和环境变量相关 ADMIN API [@nejisama](https://github.com/nejisama)
- 更新 Host 时触发健康检查的更新从异步模式修改为同步模式 [@nejisama](https://github.com/nejisama)
- 更新了 Dubbo 库，优化了 Dubbo Decode 的性能 [@zonghaishang](https://github.com/zonghaishang)
- 优化 Metrics 在 Prometheus 中的输出，使用正则过滤非法的 Key [@nejisama](https://github.com/nejisama)
- 优化 MOSN 的返回状态码 [@wangfakang](https://github.com/wangfakang)

### Bug 修复

- 修复健康检查注册回调函数时的并发冲突问题 [@nejisama](https://github.com/nejisama)
- 修复配置持久化函数没有正确处理空配置的错误 [@nejisama](https://github.com/nejisama)
- 修复 ClusterName/RouterName 过长时，以文件形式 DUMP 会失败的问题 [@nejisama](https://github.com/nejisama)
- 修复获取 XProtocol 协议时，无法正确获取协议的问题 [@wangfakang](https://github.com/wangfakang)
- 修复创建 StreamFilter 时，获取的 context 错误的问题 [@wangfakang](https://github.com/wangfakang)

## v0.12.0

### 新功能

- 支持 Skywalking [@arugal](https://github.com/arugal)
- Stream Filter 新增了一个 Receive Filter 执行的阶段，可在 MOSN 路由选择完 Host 以后，再次执行 Receive Filter [@wangfakang](https://github.com/wangfakang)
- HTTP2 支持流式 [@peacocktrain](https://github.com/peacocktrain) [@taoyuanyuan](https://github.com/taoyuanyuan)
- FeatureGate 新增接口 KnownFeatures，可输出当前 FeatureGate 状态 [@nejisama](https://github.com/nejisama)
- 提供一种协议透明的方式获取请求资源（PATH、URI、ARG），对于资源的定义由各个协议自身定义 [@wangfakang](https://github.com/wangfakang)
- 新增负载均衡算法
  - 支持 ActiveRequest LB [@CodingSinger](https://github.com/CodingSinger)
  - 支持 WRR LB [@nejisama](https://github.com/nejisama)

### 优化

- XProtocol 协议引擎优化 [@neverhook](https://github.com/neverhook)
  - 修改 XProtocol 心跳响应接口，支持协议的心跳响应可返回更多的信息
  - 优化 connpool 的心跳触发，只有实现了心跳的协议才会发心跳
- Dubbo 库依赖版本从 v1.5.0-rc1 更新到 v1.5.0 [@cch123](https://github.com/cch123)
- API 调整，HostInfo 新增健康检查相关的接口 [@wangfakang](https://github.com/wangfakang)
- 熔断功能实现优化 [@wangfakang](https://github.com/wangfakang)
- 负责均衡选择逻辑简化，同样地址的 Host 复用相同的健康检查标记 [@nejisama](https://github.com/nejisama) [@cch123](https://github.com/cch123)
- 优化 HTTP 建连逻辑，提升 HTTP 建立性能 [@wangfakang](https://github.com/wangfakang)
- 日志轮转逻辑从写日志触发，调整为定时触发 [@nejisama](https://github.com/nejisama)
- typo 调整 [@xujianhai666](https://github.com/xujianhai666) [@candyleer](https://github.com/candyleer)

### Bug 修复

- 修复 xDS 解析故障注入配置的错误 [@champly](https://github.com/champly)
- 修复 MOSN HTTP HEAD 方法导致的请求 Hold 问题 [@wangfakang](https://github.com/wangfakang)
- 修复 XProtocol 引擎对于 StatusCode 映射缺失的问题 [@neverhook](https://github.com/neverhook)
- 修复 DirectReponse 触发重试的 BUG [@taoyuanyuan](https://github.com/taoyuanyuan)

## v0.11.0

### 新功能

- 支持 Listener Filter 的扩展，透明劫持能力基于 Listener Filter 实现 [@wangfakang](https://github.com/wangfakang)
- 变量机制新增 Set 方法 [@neverhook](https://github.com/neverhook)
- 新增 SDS Client 失败时自动重试和异常处理 [@pxzero](https://github.com/pxzero)
- 完善 TraceLog，支持注入 context[@taoyuanyuan](https://github.com/taoyuanyuan)
- 新增 FeatureGate `auto_config`，当开启该 Feature 以后动态更新的配置会保存到启动配置中 [@nejisama](https://github.com/nejisama)

### 重构

- 重构 XProtocol Engine，并且重新实现了 SofaRPC 协议 [@neverhook](https://github.com/neverhook)
  - 移除了 SofaRpc Healthcheck filter，改为 xprotocol 内建的 heartbeat 实现
  - 移除了 SofaRpc 协议原本的协议转换 (protocol conv) 支持，新增了基于 stream filter 的的协议转换扩展实现能力
  - xprotocol 新增 idle free 和 keepalive
  - 协议解析优化
- 修改 HTTP2 协议的 Encode 方法参数 [@taoyuanyuan](https://github.com/taoyuanyuan)
- 精简了 LDS 接口参数 [@nejisama](https://github.com/nejisama)
- 修改了路由配置模型，废弃了`connection_manager`[@nejisama](https://github.com/nejisama)

### 优化

- 优化 Upstream 动态解析域名机制 [@wangfakang](https://github.com/wangfakang)
- 优化 TLS 封装，新增了错误日志，修改了兼容模式下的超时时间 [@nejisama](https://github.com/nejisama)
- 优化超时时间设置，使用变量机制设置超时时间 [@neverhook](https://github.com/neverhook)
- Dubbo 解析库依赖升级到 1.5.0 [@cch123](https://github.com/cch123)
- 引用路径迁移脚本新增 OS 自适应 [@taomaree](https://github.com/taomaree)

### Bug 修复

- 修复 HTTP2 协议转发时丢失 query string 的问题 [@champly](https://github.com/champly)

## v0.10.0

### 新功能

- 支持多进程插件模式
- 启动参数支持 service-meta 参数
- 支持 abstract uds 模式挂载 sds socket

### 重构

- 分离部分 mosn 基础库代码到 mosn.io/pkg 包（github.com/mosn/pkg)
- 分离部分 mosn 接口定义到 mosn.io/api 包（github.com/mosn/api)

### 优化

- 日志基础模块分离到 mosn.io/pkg，mosn 的日志实现优化
- 优化 FeatureGate
- 新增处理获取 SDS 配置失败时的处理
- CDS 动态删除 Cluster 时，会同步停止对应 Cluster 的健康检查
- sds 触发证书更新时的回调函数新增证书配置作为参数

### Bug 修复

- 修复在 SOFARPC Oneway 请求失败时，导致的内存泄漏问题
- 修复在收到非标准的 HTTP 响应时，返回 502 错误的问题
- 修复 DUMP 配置时可能存在的并发冲突
- 修复 TraceLog 统计的 Request 和 Response Size 错误问题
- 修复因为并发写连接导致写超时失效的问题
- 修复 serialize 序列化的 bug
- 修复连接读取时内存复用保留 buffer 过大导致内存占用过高的问题
- 优化 XProtocol 中 Dubbo 相关实现

## v0.9.0

### 新功能

- 支持变量机制，accesslog 修改为使用变量机制获取信息

### 重构

- 重构了包引用路径从 `sofastack.io/sofa-mosn` 变更为 `mosn.io/mosn`

## Bug 修复

- 修复 Write 连接时没有对 buf 判空的 bug
- 修复 HTTP2 Stream 计数错误的 bug
- 修复在 proxy 协程 panic 时导致的内存泄漏
- 修复在特定的场景下，读写协程卡死导致的内存泄漏
- 修复 xDS 并发处理的 bug
- make image 的产出镜像修改，修改后作为 MOSN 的示例
- 修正 SOFA RPC 的 TraceLog 中获取 CallerAPP 的字段

## v0.8.1

### 新功能

- 新增 Metrics: MOSN 处理失败的请求数

### 优化

- 通过 MMAP 提升 Metrics 共享内存的写性能
- 减少默认协程池大小，优化内存占用
- 优化日志输出

### Bug 修复

- 修复 MOSN 启动时如果存在日志文件，没有被正常轮转的 Bug

## v0.8.0

### 新功能

- 新增接口：支持连接返回当前是否可用的状态
- 管理 API 新增默认的帮助页面

### 优化

- 减少连接和请求默认的内存分配
- 优化 ConfigStore 中的机器列表信息存储
- Metrics 优化
  - SOFA RPC 的心跳请求不再记录到 Metrics 中
  - 优化共享内存方式的 Metrics 使用
- 优化配置文件读取，会忽略空文件与非 json 文件
- 优化 xDS 客户端
  - xDS 客户端修改为完全异步启动，不阻塞启动流程
  - 优化 xDS 客户端断连重试逻辑

### Bug 修复

- 修复在 TLS Inspector 模式下热升级连接迁移会失败的 Bug
- 修复日志轮转配置无法正确更新的 Bug
- 修复日志在 Fatal 级别没有正确输出日志时间的 Bug
- 修复在特定的场景下，连接的读循环会导致死循环的 Bug
- 修复 HTTP 连接计数统计错误的 Bug
- 修复关闭连接时，无法正确关闭对应 channel 的 Bug
- 修复处理 BoltV2 协议的响应时，没有正确处理 buffer 的 Bug
- 修复读写持久化配置时的并发冲突
- 修复收到响应和触发超时的并发冲突

## v0.7.0

### 新功能

- 新增 FeatureGates 支持
- 新增 Metrics: 请求在 MOSN 中处理耗时
- 支持运行时重启已经关闭的监听套接字

### 重构

- 使用的 Go 版本升级到 1.12.7
- 修改 xDS 客户端启动时机，现在会先于 MOSN 的服务启动

### Bug 修复

- 修复 RPC 请求写错误时，没有正确触发请求重置的 Bug
- 修复没有收到上游响应时产生的内存泄漏 Bug
- 修复在 HTTP 请求执行重试时，部分请求参数会丢失的 Bug
- 修复在 DNS 解析失败时，可能导致 panic 的 Bug
- 修复在 TLS Inspector 模式下连接建立时不会超时的 Bug
- prometheus 输出格式不再支持 gzip

## v0.6.0

### 新功能

- 配置新增空闲连接超时`connection_idle_timeout`字段，默认值 90s。当触发空闲连接超时后，MOSN 会主动关闭连接
- 错误日志新增 Alert 接口，输出带错误码格式的错误日志
- 支持 SDS 的方式获取 TLS 证书

### 重构

- 重构了 upstream 模块
  - 重构了内部 Cluster 实现结构
  - 更新 Host 的实现从差量更新修改为全量更新，加快更新速度
  - 重构了快照（Snapshot）的实现
  - 优化了部分内存占用情况
  - 修改了部分接口函数的参数
- 重构了 Tracing 的实现方式，支持更多的扩展

### 优化

- 优化了连接的 Metrics 统计
- 优化了 Metrics 在 prometheus 模式下的输出格式
- 优化了 IO 写协程，减少内存占用

### Bug 修复

- 修复了并发创建 Logger 时可能存在的并发冲突
- 修复了收到响应和触发超时会导致 panic 的 Bug
- 修复了 HTTP 处理连接重置时的并发 Bug
- 修复了在日志文件被删除后，无法正确轮转的 Bug
- 修复了 HTTP2 处理 goaway 时的并发 Bug

## v0.5.0

### 新功能

- 配置支持 xDS 模式和静态配置同时存在的混合模式
- 支持管理 API 可扩展注册新的 API
- 支持动态更新 StreamFilter 配置以后，可以对所有的连接生效

### 重构

- 重构了包引用路径
  - 从`github.com/alipay/sofa-mosn`变更为`sofastack.io/sofa-mosn`

### 优化

- 优化了错误日志的输出结构
- 完善了配置文件 json 解析的实现
- 优化了针对使用大 buffer 场景的内存复用
- 优化首次启动时候，对 Metrics 使用共享内存的处理

### Bug 修复

- 修复了 ProxyLogger 的日志级别无法被动态更新的 Bug
- 修复了连接的读写循环可能导致 panic 的 Bug
- 修复了同时删除多个 Cluster 时不能正确生效的 Bug
- 修复了 Metrics 中活跃请求数在并发情况下计数错误的 Bug
- 修复了 HTTP 连接在重置和收到响应并发时触发 panic 的 Bug

## v0.4.2

### 新功能

- 支持新的配置文件模型
  - 集群配置支持被设置为单独的目录
  - 路由配置支持被设置为单独的目录
  - 支持多证书配置
  - 兼容旧的配置文件模型
- 新增展示基本信息的 Metrics
  - 版本号
  - 使用的 Go 版本
  - MOSN 运行时状态
  - 监听的地址
- 支持 Metrics 的过滤
- 支持注册 MOSN 运行状态变化时的回调函数
- 支持 Request oneway
- 支持错误日志级别的批量修改、支持批量关闭 Access 日志

### 重构

- 重构了 Proxy 线程模型
  - 每个请求使用一个单独的 Goroutine 进行处理
  - 使用状态机代替回调函数，请求处理流程修改为串行
- 重构了连接池选择模型，尽量避免选择到异常的后端

### 优化

- 优化了 Metrics 输出性能
- 优化了错误日志输出
- 优化了 SOFA RPC 协议解析性能
- 扩展实现 context，在兼容标准 context 的情况下，降低嵌套层数，优化性能

### Bug 修复

- 修复了错误解析部分 json 配置的 Bug
- 修复了 HTTP 在特定场景下会触发 Goroutine 泄漏的 Bug
- 修复了 IO 写并发场景下可能导致 panic 的 Bug
- 修复了 HOST 信息没有被去重的 Bug

## v0.4.1

### 新功能

- Metrics 新增 prometheus 模式输出
- Metrics 支持 exclusion 配置
- 支持日志的动态开启、关闭，支持错误日志级别的动态调整
- HTTP 协议支持 100 continue
- 支持 Tars 协议
- 支持 SOFARPC 协议的连接在空闲时发送心跳
- 支持根据 SOFARPC 的子协议创建连接
- 支持全新的平滑升级方式
- 主动健康检查支持可扩展的实现，默认为 tcp dial
- 内存复用模块支持可扩展
- 负载均衡实现支持可扩展
- 配置文件解析方式支持可扩展，默认为 json 文件解析

### 重构

- 重构了 stream 包结构，修改部分 API 接口
- 日志模块修改为异步写日志
- 重构了 xDS 配置转换模块
- 重构了路由链的实现
- 将部分通用功能函数移动到 utils 包目录下

### 优化

- 路由匹配方式优化，支持特定场景下通过 KV 匹配
- 请求信息中记录的请求状态码统一转换为 HTTP 状态码作为标准
- 优化了 Tracer 的实现，提升 Tracer 记录的性能
- 优化了配置文件持久化的性能
- 优化了动态更新后端机器列表的性能

### Bug 修复

- 修复了 workpool 的死锁 Bug
- 修复了 HTTP2 错误处理 trailer 的 Bug
- 修复了 buffer 复用的并发问题

## v0.4.0

### 新功能

- 通过 HTTP2 支持 gRPC
- 支持 HTTP/HTTP2 协议自动识别
- 支持 SOFA RPC 协议的 Tracer
- 新增更多的路由功能
  - 支持重试策略的配置
  - 支持直接响应的策略配置
  - 支持在 HTTP Header 中添加、删除自定义的字段
  - 支持重写 HTTP 协议的 Host 和 URI
  - 支持可扩展的路由实现
- 支持 QPS 限流和基于速率的限流
- 支持故障注入
- 支持 Mixer
- 支持获取 MOSN 运行时配置

### 重构

- 重构了协议框架，支持 SOFA RPC 子协议的扩展

### 优化

- 优化了 HTTP 协议的支持，性能提升约 30%
- 优化了 HTTP2 协议的支持，性能提升约 100%
- 优化了 TCP Proxy 的实现

### Bug 修复

- 修复了平滑升级的 Bug
- 修复了 HTTP/HTTP2 协议处理的 Bug
- 修复了一些潜在的内存泄漏的 Bug

## v0.3.0

### 新功能

- 支持 Metrics 的平滑迁移
- 支持 TLS 的平滑迁移

### 优化

- 优化了 SOFARPC 协议解析的 CPU 和内存占用情况

## v0.2.1

### 新功能

- 支持单机级别的 TLS 开关
- 通过 XProtocol 协议支持 Dubbo 协议

## v0.2.0

### 新功能

- 支持带权重的路由匹配规则
- 新增 xDS 客户端实现
  - 支持 LDS
  - 支持 CDS
- 支持四层 Filter 可扩展
- 支持 TLS 配置可扩展
- 支持基于原生 epoll 的 IO 处理
- 加强协议解析的可扩展能力
- 新增 XProtocol，可通过 XProtocol 扩展协议实现

### 优化

- 实现内存复用框架，降低内存分配开销

## v0.1.0

### 新功能

- 实现了一个可编程、可扩展的网络扩展框架 MOSN
- 实现了协议框架
  - 支持 SOFARPC 协议
  - 支持 HTTP 协议
  - 支持 HTTP2 协议
- 支持基于 Stream Filters 的可扩展模式
- 支持后端集群管理与负载均衡
- 支持简单的路由匹配规则
- 支持平滑重启与平滑升级
