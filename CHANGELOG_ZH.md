# 更新日志

## v0.14.0

### 新功能

+ 支持 Istio 1.5.X [@wangfakang](https://github.com/wangfakang) [@trainyao](https://github.com/trainyao) [@champly](https://github.com/champly)
  + go-control-plane 升级到 0.9.4 版本
  + xDS 支持 ACK，新增 xDS 的 Metrics
  + 支持 Istio sourceLabels 过滤功能
  + 支持 pilot-agent 的探测接口
  + 支持更多的启动参数，适配 Istio agent 启动场景
  + gzip、strict-dns、original-dst 支持 xDS 更新
  + 移除 Xproxy 逻辑
+ Maglev 负载均衡算法支持 [@trainyao](https://github.com/trainyao)
+ 新增连接池实现，用于支持消息类请求 [@cch123](https://github.com/cch123)
+ 新增 TLS 连接切换的 Metrics [@nejisama](https://github.com/nejisama)
+ 新增 HTTP StatusCode 的 Metrics [@dengqian](https://github.com/dengqian)
+ 新增 Metrics Admin API 输出 [@dengqian](https://github.com/dengqian)
+ proxy 新增查询当前请求数的接口 [@zonghaishang](https://github.com/zonghaishang)
+ 支持 HostRewrite Header [@liangyuanpeng](https://github.com/liangyuanpeng)

### 优化

+ 升级 tars 依赖，修复在高版本 Golang 下的编译问题 [@wangfakang](https://github.com/wangfakang)
+ xDS 配置解析升级适配 Istio 1.5.x [@wangfakang](https://github.com/wangfakang)
+ 优化 proxy 的日志输出 [@wenxuwan](https://github.com/wenxuwan)
+ DNS Cache 默认时间修改为 15s [@wangfakang](https://github.com/wangfakang)
+ HTTP 参数路由匹配优化 [@wangfakang](https://github.com/wangfakang)
+ 升级 fasthttp 库 [@wangfakang](https://github.com/wangfakang)
+ 优化 Dubbo 请求转发编码 [@zonghaishang](https://github.com/zonghaishang)
+ 支持 HTTP 的请求最大 body 可配置 [@wangfakang](https://github.com/wangfakang)

### Bug 修复

+ 修复 Dubbo Decode 无法解析 attachment 的 bug [@champly](https://github.com/champly)
+ 修复 HTTP2 连接建立之前就可能创建 stream 的 bug [@dunjut](https://github.com/dunjut)
+ 修复处理 HTTP2 处理 Trailer 空指针异常 [@taoyuanyuan](https://github.com/taoyuanyuan)
+ 修复 HTTP 请求头默认不标准化处理的 bug [@nejisama](https://github.com/nejisama)
+ 修复 HTTP 请求处理时连接断开导致的 panic 异常 [@wangfakang](https://github.com/wangfakang)
+ 修复 dubbo registry 的读写锁拷贝问题 [@champly](https://github.com/champly)

## v0.13.0

### 新功能

+ 支持 Strict DNS Cluster [@dengqian](https://github.com/dengqian)
+ 支持 GZip 处理的 Stream Filter [@wangfakang](https://github.com/wangfakang)
+ Dubbo 服务发现完成 Beta 版本 [@cch123](https://github.com/cch123)
+ 支持单机故障隔离的 Stream Filter [@NeGnail](https://github.com/NeGnail)
+ 集成 Sentinel 限流能力 [@ansiz](https://github.com/ansiz)

### 优化

+ 优化 EDF LB 的实现，使用 EDF 重新实现 WRR LB [@CodingSinger](https://github.com/CodingSinger)
+ 配置获取 ADMIN API 优化，新增 Features 和环境变量相关 ADMIN API [@nejisama](https://github.com/nejisama)
+ 更新 Host 时触发健康检查的更新从异步模式修改为同步模式 [@nejisama](https://github.com/nejisama)
+ 更新了 Dubbo 库，优化了 Dubbo Decode 的性能 [@zonghaishang](https://github.com/zonghaishang)
+ 优化 Metrics 在 Prometheus 中的输出，使用正则过滤非法的 Key [@nejisama](https://github.com/nejisama)
+ 优化 MOSN 的返回状态码 [@wangfakang](https://github.com/wangfakang)

### Bug 修复

+ 修复健康检查注册回调函数时的并发冲突问题 [@nejisama](https://github.com/nejisama)
+ 修复配置持久化函数没有正确处理空配置的错误 [@nejisama](https://github.com/nejisama)
+ 修复 ClusterName/RouterName 过长时，以文件形式 DUMP 会失败的问题 [@nejisama](https://github.com/nejisama)
+ 修复获取 XProtocol 协议时，无法正确获取协议的问题 [@wangfakang](https://github.com/wangfakang)
+ 修复创建 StreamFilter 时，获取的 context 错误的问题 [@wangfakang](https://github.com/wangfakang)


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
