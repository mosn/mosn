# 更新日志

## v0.10.0

### 新功能

- 支持多进程插件模式
- 启动参数支持service-meta参数
- 支持abstract uds模式挂载sds socket

### 重构

- 分离部分mosn基础库代码到mosn.io/pkg包（github.com/mosn/pkg)
- 分离部分mosn接口定义到mosn.io/api包（github.com/mosn/api)

### 优化

- 日志基础模块分离到mosn.io/pkg，mosn的日志实现优化
- 优化FeatureGate
- 新增处理获取SDS配置失败时的处理
- CDS动态删除Cluster时，会同步停止对应Cluster的健康检查
- sds触发证书更新时的回调函数新增证书配置作为参数

### Bug 修复

- 修复在SOFARPC Oneway请求失败时，导致的内存泄漏问题
- 修复在收到非标准的HTTP响应时，返回502错误的问题
- 修复DUMP配置时可能存在的并发冲突
- 修复TraceLog统计的Request和Response Size错误问题
- 修复因为并发写连接导致写超时失效的问题
- 修复serialize序列化的bug
- 修复连接读取时内存复用保留buffer过大导致内存占用过高的问题
- 优化XProtocol中Dubbo相关实现

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

- 新增Metrics: MOSN处理失败的请求数

### 优化

- 通过MMAP提升Metrics共享内存的写性能
- 减少默认协程池大小，优化内存占用
- 优化日志输出

### Bug修复

- 修复MOSN启动时如果存在日志文件，没有被正常轮转的Bug

## v0.8.0

### 新功能

- 新增接口: 支持连接返回当前是否可用的状态
- 管理API新增默认的帮助页面

### 优化

- 减少连接和请求默认的内存分配
- 优化ConfigStore中的机器列表信息存储
- Metrics优化
  - SOFA RPC的心跳请求不再记录到Metrics中
  - 优化共享内存方式的Metrics使用
- 优化配置文件读取，会忽略空文件与非json文件
- 优化xDS客户端
  - xDS客户端修改为完全异步启动，不阻塞启动流程
  - 优化xDS客户端断连重试逻辑

### Bug修复

- 修复在TLS Inspector模式下热升级连接迁移会失败的Bug
- 修复日志轮转配置无法正确更新的Bug
- 修复日志在Fatal级别没有正确输出日志时间的Bug
- 修复在特定的场景下，连接的读循环会导致死循环的Bug
- 修复HTTP连接计数统计错误的Bug
- 修复关闭连接时，无法正确关闭对应channel的Bug
- 修复处理BoltV2协议的响应时，没有正确处理buffer的Bug
- 修复读写持久化配置时的并发冲突
- 修复收到响应和触发超时的并发冲突

## v0.7.0

### 新功能

- 新增FeatureGates支持
- 新增Metrics: 请求在MOSN中处理耗时
- 支持运行时重启已经关闭的监听套接字

### 重构

- 使用的Go版本升级到1.12.7
- 修改xDS客户端启动时机，现在会先于MOSN的服务启动

### Bug修复

- 修复RPC请求写错误时，没有正确触发请求重置的Bug
- 修复没有收到上游响应时产生的内存泄漏Bug
- 修复在HTTP请求执行重试时，部分请求参数会丢失的Bug
- 修复在DNS解析失败时，可能导致panic的Bug
- 修复在TLS Inspector模式下连接建立时不会超时的Bug
- prometheus输出格式不再支持gzip

## v0.6.0

### 新功能

- 配置新增空闲连接超时`connection_idle_timeout`字段，默认值90s。当触发空闲连接超时后，MOSN会主动关闭连接
- 错误日志新增Alert接口，输出带错误码格式的错误日志
- 支持SDS的方式获取TLS证书

### 重构

- 重构了upstream模块
  - 重构了内部Cluster实现结构
  - 更新Host的实现从差量更新修改为全量更新，加快更新速度
  - 重构了快照（Snapshot）的实现
  - 优化了部分内存占用情况
  - 修改了部分接口函数的参数
- 重构了Tracing的实现方式，支持更多的扩展

### 优化

- 优化了连接的Metrics统计
- 优化了Metrics在prometheus模式下的输出格式
- 优化了IO写协程，减少内存占用

### Bug修复

- 修复了并发创建Logger时可能存在的并发冲突
- 修复了收到响应和触发超时会导致panic的Bug
- 修复了HTTP处理连接重置时的并发Bug
- 修复了在日志文件被删除后，无法正确轮转的Bug
- 修复了HTTP2处理goaway时的并发Bug

## v0.5.0

### 新功能

- 配置支持xDS模式和静态配置同时存在的混合模式
- 支持管理API可扩展注册新的API
- 支持动态更新StreamFilter配置以后，可以对所有的连接生效

### 重构

- 重构了包引用路径
  - 从`github.com/alipay/sofa-mosn`变更为`sofastack.io/sofa-mosn`

### 优化

- 优化了错误日志的输出结构
- 完善了配置文件json解析的实现
- 优化了针对使用大buffer场景的内存复用
- 优化首次启动时候，对Metrics使用共享内存的处理

### Bug修复

- 修复了ProxyLogger的日志级别无法被动态更新的Bug
- 修复了连接的读写循环可能导致panic的Bug
- 修复了同时删除多个Cluster时不能正确生效的Bug
- 修复了Metrics中活跃请求数在并发情况下计数错误的Bug
- 修复了HTTP连接在重置和收到响应并发时触发panic的Bug

## v0.4.2

### 新功能

- 支持新的配置文件模型
  - 集群配置支持被设置为单独的目录
  - 路由配置支持被设置为单独的目录
  - 支持多证书配置
  - 兼容旧的配置文件模型
- 新增展示基本信息的Metrics
  - 版本号
  - 使用的Go版本
  - MOSN运行时状态
  - 监听的地址
- 支持Metrics的过滤
- 支持注册MOSN运行状态变化时的回调函数
- 支持Request oneway
- 支持错误日志级别的批量修改、支持批量关闭Access日志

### 重构

- 重构了Proxy线程模型
  - 每个请求使用一个单独的Goroutine进行处理
  - 使用状态机代替回调函数，请求处理流程修改为串行
- 重构了连接池选择模型，尽量避免选择到异常的后端

### 优化

- 优化了Metrics输出性能
- 优化了错误日志输出
- 优化了SOFA RPC协议解析性能
- 扩展实现context，在兼容标准context的情况下，降低嵌套层数，优化性能 

### Bug修复

- 修复了错误解析部分json配置的Bug
- 修复了HTTP在特定场景下会触发Goroutine泄漏的Bug
- 修复了IO写并发场景下可能导致panic的Bug
- 修复了HOST信息没有被去重的Bug

## v0.4.1

### 新功能

- Metrics新增prometheus模式输出
- Metrics支持exclusion配置
- 支持日志的动态开启、关闭，支持错误日志级别的动态调整
- HTTP协议支持100 continue
- 支持Tars协议
- 支持SOFARPC协议的连接在空闲时发送心跳
- 支持根据SOFARPC的子协议创建连接
- 支持全新的平滑升级方式
- 主动健康检查支持可扩展的实现，默认为tcp dial
- 内存复用模块支持可扩展
- 负载均衡实现支持可扩展
- 配置文件解析方式支持可扩展，默认为json文件解析

### 重构

- 重构了stream包结构，修改部分API接口
- 日志模块修改为异步写日志
- 重构了xDS配置转换模块
- 重构了路由链的实现
- 将部分通用功能函数移动到utils包目录下

### 优化

- 路由匹配方式优化，支持特定场景下通过KV匹配
- 请求信息中记录的请求状态码统一转换为HTTP状态码作为标准
- 优化了Tracer的实现，提升Tracer记录的性能
- 优化了配置文件持久化的性能
- 优化了动态更新后端机器列表的性能

### Bug修复

- 修复了workpool的死锁Bug
- 修复了HTTP2错误处理trailer的Bug
- 修复了buffer复用的并发问题

## v0.4.0

### 新功能

- 通过HTTP2支持gRPC
- 支持HTTP/HTTP2协议自动识别
- 支持SOFA RPC协议的Tracer
- 新增更多的路由功能
  - 支持重试策略的配置
  - 支持直接响应的策略配置
  - 支持在HTTP Header中添加、删除自定义的字段
  - 支持重写HTTP协议的Host和URI
  - 支持可扩展的路由实现
- 支持QPS限流和基于速率的限流
- 支持故障注入
- 支持Mixer
- 支持获取MOSN运行时配置

### 重构

- 重构了协议框架，支持SOFA RPC子协议的扩展

### 优化

- 优化了HTTP协议的支持，性能提升约30%
- 优化了HTTP2协议的支持，性能提升约100%
- 优化了TCP Proxy的实现

### Bug修复

- 修复了平滑升级的Bug
- 修复了HTTP/HTTP2协议处理的Bug
- 修复了一些潜在的内存泄漏的Bug

## v0.3.0

### 新功能

- 支持Metrics的平滑迁移
- 支持TLS的平滑迁移

### 优化

- 优化了SOFARPC协议解析的CPU和内存占用情况

## v0.2.1

### 新功能

- 支持单机级别的TLS开关
- 通过XProtocol协议支持Dubbo协议

## v0.2.0

### 新功能

- 支持带权重的路由匹配规则
- 新增xDS客户端实现
  - 支持LDS
  - 支持CDS
- 支持四层Filter可扩展
- 支持TLS配置可扩展
- 支持基于原生epoll的IO处理
- 加强协议解析的可扩展能力
- 新增XProtocol，可通过XProtocol扩展协议实现

### 优化

- 实现内存复用框架，降低内存分配开销

## v0.1.0

### 新功能

- 实现了一个可编程、可扩展的网络扩展框架MOSN
- 实现了协议框架
  - 支持SOFARPC协议
  - 支持HTTP协议
  - 支持HTTP2协议
- 支持基于Stream Filters的可扩展模式
- 支持后端集群管理与负载均衡
- 支持简单的路由匹配规则
- 支持平滑重启与平滑升级
