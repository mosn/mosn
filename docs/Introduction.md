# MOSN Introduction

## MOSN 是什么
MOSN是一款采用 Golang 开发的Service Mesh数据平面代理，功能和定位类似Envoy，旨在提供分布式，模块化，可观察，智能化的代理能力。MOSN支持Envoy和Istio的API，可以和Istio集成。SOFAMesh中，我们使用MOSN替代Envoy。
初始版本由蚂蚁金服和阿里大文娱UC事业部携手贡献，期待社区一起来参与后续开发，共建一个开源精品项目。

# MOSN 框架总览
MOSN 由 NET/IO、Protocol、Stream、Proxy 四个层次组成，其中

+ NET/IO 用于底层的字节流传输
+ Protocol 用于协议的 decode/encode
+ Stream 用于封装请求和响应，在一个 conn 上做连接复用
+ Proxy 做 downstream 和 upstream 之间 stream 的转发
下图为构成 MOSN 的模块:

![modules](design/resource/MosnModules.png)

+ Starter, Server, Listener, Config, XDS 为 MOSN 启动模块，用于完成 MOSN 的运行
+ 最左侧的 Hardware, NET/IO, Protocol, Stream, Proxy 为MOSN 核心模块，
  用来完成 Service MESH 的核心功能，后面会专题单独介绍
+ Router 为 MOSN 的核心路由模块，支持的功能包括：
    + VirtualHost 形式的路由表
    + 基于 subset 的子集群路由匹配
    + 路由重试以及重定向功能
+ Upstream 为后端管理模块，支持的功能包括：
    + Cluster 动态更新
    + Host 动态更新
    + 对 Cluster 的 主动/被动 健康检查
    + 熔断机制
    + CDS/EDS 对接能力
+ Metrics 模块可对协议层的数据做记录和追踪
+ LoadBalance 当前支持 Random, RR, SWRR, Subset LB, 等负载均衡算法
+ Mixer, FlowControl, Lab, Admin 模块为待开发模块

## MOSN 工作模式
+ MOSN 工作流程
下图展示的是使用 Sidecar 方式部署运行 MSON 的示意图，Service 和 MOSN 分别部署在同机部署的 Pod 上，
您可以在配置文件中设置 MOSN 的上游和下游协议，协议在 HTTP、HTTP2.0、以及SOFA RPC 中选择，未来还将支持
DUBBO, HSF 等

![MOSN 工作流程图](design/resource/MosnWorkFlow.png)

+ MOSN 内部数据流
+ MOSN 内部数据流如下图所示

<img src="design/resource/MosnDataFlow.png" width = "300" height = "300" align="middle" />

+ NET/IO 监测连接和数据包的到来
+ Protocol 对数据包进行检测，并使用对应协议做 decode 处理
+ Streaming 对 decode 的数据包做二次封装为stream
+ Protocol 对封装的 stream 做 proxy

## MOSN 协程模型
MOSN 通过开关，允许使用如下两种线程模型
+ epoll + 事件驱动编程模型
+ golang 默认的 epoll + 协程阻塞编程模型

## MOSN 扩展机制
+ 支持 协议扩展
+ 支持 NetworkFilter 扩展
+ 支持 StreamFilter 扩展
## MOSN 历史版本
当前发布了

+ 0.1.0
+ 0.2.0  

两个版本

## MOSN 开发团队
+ 蚂蚁金服系统部网络团队
+ 蚂蚁金服中间件团队
+ UC 大文娱团队

## MOSN 系列文章编写团队以及文章认领
todo

## 获取相关帮助
* [社区](https://github.com/alipay/mosn/issues)

todo