# MOSN Project

[![Build Status](https://travis-ci.org/alipay/sofa-mosn.svg?branch=master)](https://travis-ci.org/alipay/sofa-mosn)
[![codecov](https://codecov.io/gh/alipay/sofa-mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/alipay/sofa-mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/alipay/sofa-mosn)](https://goreportcard.com/report/github.com/alipay/sofa-mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

MOSN(Modular Observable Smart Network), 是一款采用 GoLang 开发的 Service Mesh 数据平面代理，
功能和定位类似 [Envoy](https://www.envoyproxy.io/)，旨在提供分布式，模块化，可观察，智能化的代理能力。
MOSN 支持 Envoy 和 [Istio](https://istio.io/) 的 API，可以和 Istio 集成，在 [SOFAMesh](https://github.com/alipay/sofa-mesh) 中，我们使用 MOSN 替代 Envoy。
MOSN 初始版本由蚂蚁金服和阿里大文娱UC事业部携手贡献，期待社区一起来参与后续开发，共建一个开源精品项目。

### 下载

+ 可以通过`go get -u sofastack.io/sofa-mosn`，或者使用git下载到`$GOPATH/src/sofastack.io/sofa-mosn`目录下。
+ 如果你需要使用0.5.0之前的代码，你可能需要执行`bash transfer_path.sh`进行路径的适配
  + 如果是在Linux下，需要修改一下`transfer_path.sh`中的`SED_CMD`，具体修改见脚本注释

## 研发状态

我们目前已经在0.4.0版本的基础之上，进行了较多性能与稳定性上的开发，初步预计将在2019年5月底发出0.5.0版本。0.5.0版本将是一个在蚂蚁内部验证过的可用版本。
从2019年3月起，在0.5.0版本发出来之前，我们会在每个月底出一个0.4.X的版本，同步最新的改动。

## [MOSN 详细介绍](docs/Introduction.md)

## 核心能力

+ Istio集成
    + 集成 Istio 1.0 版本与 V4 API，可基于全动态资源配置运行
+ 核心转发
    + 自包含的网络服务器
    + 支持 TCP 代理
    + 支持 TProxy 模式
+ 多协议
    + 支持 HTTP/1.1，HTTP/2
    + 支持 SOFARPC
    + 支持 Dubbo 协议（基于XProtocol支持）
+ 核心路由
    + 支持 Virtual Host 路由
    + 支持 Headers/URL/Prefix 路由
    + 支持基于 Host Metadata 的 Subset 路由
    + 支持重试
+ 后端管理&负载均衡
    + 支持连接池
    + 支持熔断
    + 支持后端主动健康检查
    + 支持 Random/RR 等负载策略
    + 支持基于 Host Metadata 的 Subset 负载策略
+ 可观察性
    + 观察网络数据
    + 观察协议数据
+ TLS
    + 支持 HTTP/1.1 on TLS
    + 支持 HTTP/2.0 on TLS
    + 支持 SOFARPC on TLS
+ 进程管理
    + 支持平滑 reload
    + 支持平滑升级
+ 扩展能力
    + 支持自定义私有协议
    + 支持在 TCP IO 层，协议层面加入自定义扩展
    
## 快速开始
* [参考这里](docs/quickstart/Setup.md) 

## 文档
* [目录](docs/Catalog.md)

## 社区
* [Issues](https://github.com/alipay/sofa-mosn/issues)

## 版本
* [变更记录](docs/CHANGELOG.md)

## 贡献
+ [代码贡献](docs/develop/CONTRIBUTING.md) 
+ MOSN仍处在初级阶段，有很多能力需要补全，所以我们欢迎所有人参与进来与我们一起共建。
   
## 致谢
感谢Google、IBM、Lyft创建了Envoy、Istio体系，并开源了优秀的项目，使MOSN有了非常好的参考，使我们能快速落地自己的想法。
