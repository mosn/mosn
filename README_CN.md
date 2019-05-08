# MOSN Project

[![Build Status](https://travis-ci.org/alipay/sofa-mosn.svg?branch=master)](https://travis-ci.org/alipay/sofa-mosn)
[![codecov](https://codecov.io/gh/alipay/sofa-mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/alipay/sofa-mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/alipay/sofa-mosn)](https://goreportcard.com/report/github.com/alipay/sofa-mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

MOSN(Modular Observable Smart Network), 是一款采用 GoLang 开发的 Service Mesh 数据平面代理，旨在为服务提供分布式、模块化、可观察、智能化的代理能力。 MOSN 通过 XDS API 与 SOFAMesh 集成，同时 MOSN 可以作为独立的4、7层负载均衡使用，未来 MOSN 将支持更多云原生场景，并支持 nginx 的核心转发功能。

## 研发状态

我们目前已经在0.4.0版本的基础之上，进行了较多性能与稳定性上的开发，初步预计将在2019年5月底发出0.5.0版本。0.5.0版本将是一个在蚂蚁内部验证过的可用版本。
从2019年3月起，在0.5.0版本发出来之前，我们会在每个月底出一个0.4.X的版本，同步最新的改动。

## [MOSN 详细介绍](docs/Introduction.md)

## 核心能力

+ 集成 SOFAMesh，通过XDS api对接支持全动态资源配置运行
+ 支持 TCP 代理、HTTP 协议、多种 RPC 代理能力
+ 支持丰富的路由特性
+ 支持可靠后端管理，负载均衡能力
+ 支持网络层、协议层的可观察性
+ 支持多种协议基于 TLS 运行，支持 mTLS
+ 支持丰富的扩展能力，提供高度自定义扩展能力
+ 支持无损平滑升级
    
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
