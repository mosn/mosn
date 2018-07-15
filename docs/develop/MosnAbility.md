# MOSN 当前能力具备能力说明
当前 MOSN 具备如下的能力：

+ NET/IO 
    + 屏蔽IO处理细节
    + 定义网络链接生命周期
    + 定义可编程的网络模型，核心方法，监控指标
    + 定义可扩展的插件机制
    + 定义适用于client/server端的能力
    + 基于golang net包实现
    
+ PROTOCOL
    + 定义编解码核心数据结构
        + 三段式：Headers + Data + Trailers
    + 定义编/解码器核心接口
    + 定义扩展机制接受解码数据
    + 提供通用的编/解码引擎
        + 编码：对业务数据进行编码并根据控制指令发送数据
        + 解码：对IO数据进行解码并通过扩展机制通知订阅方

+ STREAMING
    + STREAM: 为网络协议请求/响应提供可编程的抽象载体
        + 考虑PING-PONG，PIPELINE，分帧STREAM三种典型流程特征
    + 定义STREAM生命周期，核心事件
    + 定义STREAM层编/解码核心接口
    + 核心数据结构复用PROTOCOL层
    + 定义可扩展的插件机制
    + 定义连接池模型
    + 编/解码过程中需处理上层传入的全局状态定义
    + 不同协议根据自身协议流程封装STREAM细节

+ PROXY
    + 基于STREAM提供多协议转发能力
    + 提供云部署亲和的后端管理能力
    + 提供负载均衡，健康检查等核心能力
    + 提供可配置的路由转发能力
    + 维护上/下游核心统计指标
    + 基于全局状态定义的请求拦截，由STREAM进行协议转换

+ 协议支持
    + SOFARPC
        + 基于Codec扩展实现BOLT v1, v2，TR协议编解码
        + 将SOFA RPC请求/响应分解为headers，body两部分
        + Tr协议body中的request id并入headers
        + 基于Stream扩展实现sofa rcp协议流程，链接池
        + 链接池只有一个client，无扩容机制
    + HTTP
        + 使用 fasthttp 开源软件
    + HTTP2
        + 使用go自带的 http2
    + [注] mosn前期重点在做sofarpc支持，对http和http2使用了开源版本，未经过性能优化
    
+ SOFARPC – 心跳处理
    + SrvA 2 Mesh
        + SrvA根据链接空闲状态开始/停止心跳发起
        + Mesh收到心跳请求后根据后端cluster健康度对Srv A进行响应
    + Mesh 2 Mesh
        + Mesh定时异步对后端Mesh进行健康检查
        + 后端Mesh收到健康检查请求后做带缓存的透传
    + Mesh 2 Srv B
        + 后端Mesh缓存失效后透传健康检查到Srv B

+ Mesh间通信-HTTP2
    + HTTP 2特性
        + 分帧多路复用
        + 二进制传输
        + 服务器推送
        + 头信息压缩
        + 数据流优先级
        + PING帧探活

+ 复杂路由支持
    + 通过子集群匹配与header过滤组合使用支持复杂的分流需求
    + 通过header kv匹配决定目标路由
        + Value匹配规则支持正则表达式
    + 通过子集群匹配灵活调度流量到统一集群内的不同子集群
        + 路由声明
            + 通过一组kv定义路由过滤规则
            + 通过路由权重调整各子集群流量比例
        + 后端子集群声明
            + 后端可携带多组自定义metadata，每组metadata为kv形式
            + 通过一组metadata keys归纳一个子集群，用于与路由声明中定义的keys对应
            + 一个集群内部可声明多个子集群

+ 网络/协议层扩展机制
    + 通过实现4层网络Filter在4层处理阶段注入扩展点
        + Connection Read/Write Filter
        + Listener Event Handler
    + 通过实现协议Stream Filter在协议处理阶段注入扩展点
        + Stream Encode/Decode Filter
        + Stream Event Handler
