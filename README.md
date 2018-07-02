# MOSN Project

![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

MOSN是一款基于 Golang 实现的Service Mesh数据平面转发代理，旨在提供模块化，可观察，智能化的代理能力，由蚂蚁金服公司开源贡献。

## 背景
ServiceMesh衍生于云原生，微服务等技术生态并得到快速发展，越来越多公司意识到Mesh技术的便利性和重要性，并纷纷加入了生态建设。蚂蚁在调研探索了业界开源项目的基础上，进行了多轮验证试错，最终选择自研数据平面，并对社区开源。

## 核心能力

### 支持Istio 0.8
+ 集成 Istio 0.8 版本 Pilot V2 api，可基于全动态资源配置运行

### 核心转发
+ 支持TCP代理
+ 支持Tproxy模式

### 多协议
+ 支持Http 1.1，Http2
+ 支持SofaRpc
+ 支持Dubbo（研发中）

### 核心路由
+ 支持virtual host路由
+ 支持headers/url/prefix路由
+ 支持基于host metadata的subset路由

### 后端管理
+ 支持cluster管理
+ 支持链接池
+ 支持后端主动健康检查
+ 支持random/rr等负载策略
+ 支持基于host metadata的subset负载策略

### mTls
+ 支持Http 1.1 on Tls
+ 支持Http2 on Tls
+ 支持SofaRpc on Tls

### 进程管理
+ 支持平滑reload
+ 支持平滑升级

### 扩展能力
+ 支持自定义私有协议
+ 支持在tcp io层，协议层面加入自定义扩展

## 快速开始
* [样例工程](mosn-samples)
  * [配置标准Http协议Mesher](samples/http-sample)
  * [配置SofaRpc协议Mesher](samples/sofarpc-sample)
 
## 社区
* [Issues](https://github.com/alipay/mosn/issues)

## 贡献
* [代码贡献](./CONTRIBUTING.md) 
+ MOSN借鉴了ENVOY核心思路，根据我们的需求在多协议支持，路由，进程管理等方面进行了调整。目前项目仍处在初级阶段，仍有很多能力协议补全，很多bug需要修复，欢迎所有人提交代码。我们欢迎您参与但不限于如下方面：
   + 核心路由功能点补全
   + Outlier detection
   + Tracing支持
   + 流控
   + 性能