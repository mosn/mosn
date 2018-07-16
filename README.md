# MOSN Project

![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

MOSN是一款基于 Golang 实现的Service Mesh数据平面代理，旨在提供分布式，模块化，可观察，智能化的代理能力，由蚂蚁金服公司开源贡献。

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

## 快速开始
* [样例工程](samples)
  * [配置标准Http协议Mesher](samples/http-sample)
  * [配置SOFARPC协议Mesher](samples/sofarpc-sample)
* 基于Golang 1.9.2研发，使用dep进行依赖管理

## 相关文档
* [Docs](http://www.sofastack.tech/sofa-mesh/docs/Home)

## ISSUES
* [Issues](https://github.com/alipay/sofa-mosn/issues)

## 贡献
+ [代码贡献](./CONTRIBUTING.md) 
+ MOSN仍处在初级阶段，有很多能力需要补全，很多bug需要修复，欢迎所有人提交代码。我们欢迎您参与但不限于如下方面：
   + 核心路由功能点补全
   + Outlier detection
   + Tracing支持
   + HTTP/1.x, HTTP/2.0性能优化
   + 流控
   
## 致谢
感谢Google，Lyft创建了ServiceMesh体系，并开源了优秀的项目，使MOSN有了非常好的参考，使我们能快速落地自己的想法
