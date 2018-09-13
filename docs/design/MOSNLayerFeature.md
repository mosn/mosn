
### NET/IO 层提供了 IO 读写的封装以及可扩展的 IO 事件订阅机制，具有下面的一些设计特性:
+ 屏蔽IO处理细节
+ 自定义网络链接生命周期的管理
+ 可编程的网络模型，核心方法，监控指标
+ 可扩展的插件机制
+ 适用于client/server端的能力

### Protocol 层提供了根据不同协议对数据进行序列化/反序列化的处理能力，具有如下的一些设计特性:
+ 定义了编解码核心数据结构
+ 三段式：Headers + Data + Trailers
+ 使用统一的 编/解码器核心接口
+ 提供通用的编/解码引擎
  + 编码：对业务数据进行编码并根据控制指令发送数据
  + 解码：对IO数据进行解码并通过扩展机制通知订阅方
### 设计了一种通用的Mesh通信协议: X-Protocol，提供了不同程度的解包能力，包括：
  + 轻度解包： Multiplexing, Accesslog, Rate limit, Curcuit Breakers
  + 中度解包： Tracing ， RequestRouting
  + 重度解包： ProtocolConvert
### Streaming 层提供向上的协议一致性，负责 stream 的生命周期，管理 Client / Server 模式的请求流行为，对 Client 端stream 提供池化机制等，具有如下一些设计特性：
  + 定义STREAM层编/解码核心接口，提供可扩展的插件机制
  + 不同协议根据自身协议流程封装STREAM细节
  + 支持多种通信模型，包括：PING-PONG，PIPELINE，分帧STREAM三种典型流程特征
### Proxy 层提供路由选择，负载均衡等的能力，做数据流之间的转发，具有如下的一些设计特性：
  + 可扩展性，提供上下游可配置的多协议转发能力
  + 具备负载均衡，健康检查，熔断等后端管理能力，云端部亲和性
  + Metrics 能力，可统计上下游的路由转发指标
