# MOSN

![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

MOSN是一款采用 Golang 开发的Service Mesh数据平面代理，功能和定位类似Envoy，旨在提供分布式，模块化，可观察，智能化的代理能力。MOSN支持Envoy和Istio的API，可以和Istio集成。Sofa Mesh中，我们使用MOSN替代Envoy。

初始版本由蚂蚁金服技术团队开源贡献，期待社区一起来参与MOSN项目的后续开发，共建一个开源精品项目。

## 核心能力

+ Istio集成
    + 集成 Istio 0.8 版本 Pilot V2 API，可基于全动态资源配置运行
+ 核心转发
    + 自包含的网络服务器
    + 支持TCP代理
    + 支持TProxy模式
+ 多协议
    + 支持HTTP/1.1，HTTP/2
    + 支持SOFARPC
    + 支持Dubbo协议（开发中）
+ 核心路由
    + 支持virtual host路由
    + 支持headers/url/prefix路由
    + 支持基于host metadata的subset路由
    + 支持重试
+ 后端管理&负载均衡
    + 支持连接池
    + 支持熔断
    + 支持后端主动健康检查
    + 支持random/rr等负载策略
    + 支持基于host metadata的subset负载策略
+ 可观察性
    + 观察网络数据
    + 观察协议数据
+ TLS
    + 支持HTTP/1.1 on TLS
    + 支持HTTP/2 on TLS
    + 支持SOFARPC on TLS
+ 进程管理
    + 支持平滑reload
    + 支持平滑升级
+ 扩展能力
    + 支持自定义私有协议
    + 支持在TCP IO层，协议层面加入自定义扩展
    
## MOSN 迭代版本特性
+ [0.2.0](../features/0.2.0/0-2-0-list.md) 
## 架构设计
* [参考这里](../design/README.md)
## 快速开始

* [参考这里](../develop/quickstart.md) 

## 文档
* [相关使用文档](../reference/README.md)
* [部分开发文档](../develop/README.md)

## 社区
* [Issues](https://github.com/alipay/mosn/issues)

## 贡献
+ [代码贡献](../develop/CONTRIBUTING.md) 
+ MOSN仍处在初级阶段，有很多能力需要补全，很多bug需要修复，欢迎所有人提交代码。我们欢迎您参与但不限于如下方面：
   + 核心路由功能点补全
   + http/http2 性能优化
   + Outlier Detection
   + Tracing支持
   + 流控
   + 性能
   
## 致谢
感谢Google，IBM，Lyft创建了Istio和Envoy体系，并开源这些优秀的项目，使MOSN有了非常好的参考，帮助我们能快速落地自己的想法。特此鸣谢！
