<p align="center">
<img src="https://raw.githubusercontent.com/mosn/community/master/icons/png/mosn-labeled-horizontal.png" width="350" title="MOSN Logo" alt="MOSN logo">
</p>

[![Build Status](https://travis-ci.com/mosn/mosn.svg?branch=master)](https://travis-ci.com/mosn/mosn)
[![codecov](https://codecov.io/gh/mosn/mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/mosn/mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/mosn/mosn)](https://goreportcard.com/report/github.com/mosn/mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

[English](README.md)

MOSN 是一款使用 Go 语言开发的网络代理软件，作为云原生的网络数据平面，旨在为服务提供多协议，模块化，智能化，安全的代理能力。MOSN 是 Modular Open Smart Network-proxy 的简称。MOSN 可以与任何支持 xDS API 的 Service Mesh 集成，亦可以作为独立的四、七层负载均衡，API Gateway，云原生 Ingress 等使用。

## 功能

MOSN 作为一款开源的网络代理，具有以下核心功能：

- 通过 xDS API 对接 Service Mesh，支持全动态资源配置运行
- 支持 TCP 代理、HTTP 协议、多种 RPC 代理能力
- 支持丰富的路由特性
- 支持可靠后端管理，负载均衡能力
- 支持网络层、协议层的可观察性
- 支持多种协议基于 TLS 运行，支持 mTLS
- 支持丰富的扩展能力，提供高度自定义扩展能力
- 支持无损平滑升级

## 下载安装

使用 `go get -u mosn.io/mosn` 命令或者将项目代码克隆到 `$GOPATH/src/mosn.io/mosn` 目录中。

**注意事项**

- 如果您想使用 v0.8.1 以前的版本，需要使用 `transfer_path.sh` 命令修复代码包导入问题。
- 如果您使用的是 Linux 系统，需要修改 `transfer_path.sh` 脚本中的 `SED_CMD` 的变量，请参阅脚本中的注释。

## 文档

- [MOSN 官网](https://mosn.io)
- [MOSN 版本更新日志](CHANGELOG_ZH.md)

## 贡献

请参阅[贡献者指南](CONTRIBUTING_ZH.md)。

## 社区

请访问 <https://github.com/mosn/community> 了解更多社区信息。

使用钉钉扫描下面的二维码加入 MOSN 用户交流群。

<p align="center">
<img src="https://gw.alipayobjects.com/mdn/rms_91f3e6/afts/img/A*NyEzRp3Xq28AAAAAAAAAAABkARQnAQ" width="150" title="MOSN用户交流群" alt="MOSN 用户交流群">
</p>

## Landscapes

<p align="center">
<img src="https://landscape.cncf.io/images/left-logo.svg" width="150"/>&nbsp;&nbsp;<img src="https://landscape.cncf.io/images/right-logo.svg" width="200"/>
<br/><br/>
MOSN enriches the <a href="https://landscape.cncf.io/landscape=observability-and-analysis&license=apache-license-2-0">CNCF CLOUD NATIVE Landscape.</a>
</p>