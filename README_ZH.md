<p align="center">
<img src="https://raw.githubusercontent.com/mosn/community/master/icons/png/mosn-labeled-horizontal.png" width="350" title="MOSN Logo" alt="MOSN logo">
</p>

[![Build Status](https://travis-ci.com/mosn/mosn.svg?branch=master)](https://travis-ci.com/mosn/mosn)
[![codecov](https://codecov.io/gh/mosn/mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/mosn/mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/mosn/mosn)](https://goreportcard.com/report/github.com/mosn/mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

[English](README.md)

MOSN（Modular Open Smart Network）是一款主要使用 Go 语言开发的云原生网络代理平台，由蚂蚁集团开源并经过双11大促几十万容器的生产级验证。
MOSN 为服务提供多协议、模块化、智能化、安全的代理能力，融合了大量云原生通用组件，同时也可以集成 Envoy 作为网络库，具备高性能、易扩展的特点。
MOSN 可以和 Istio 集成构建 Service Mesh，也可以作为独立的四、七层负载均衡，API Gateway、云原生 Ingress 等使用。

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

- [MOSN 官网](https://mosn.io/zh)
- [MOSN 版本更新日志](CHANGELOG_ZH.md)

## 贡献

请参阅[贡献者指南](CONTRIBUTING_ZH.md)。

## 合作伙伴

合作伙伴参与 MOSN 合作开发，使 MOSN 变得更好。

<div>
<table>
  <tbody>
  <tr></tr>
    <tr>
      <td align="center"  valign="middle">
        <a href="https://www.antfin.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/ant.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="https://www.aliyun.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/aliyun.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.zhipin.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/bosszhipin.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.dmall.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/duodian.png">
        </a>
      </td>
      </tr><tr></tr>
      <tr>
      <td align="center" valign="middle">
        <a href="https://www.kanzhun.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/kanzhun.png">
        </a>
      </td>
    </tr>
    <tr></tr>
  </tbody>
</table>
</div>

## 终端用户

以下是 MOSN 的用户。请在[此处](https://github.com/mosn/community/issues/8)登记并提供反馈来帮助 MOSN 做的更好。

<div>
<table>
  <tbody>
  <tr></tr>
    <tr>
      <td align="center"  valign="middle">
        <a href="https://www.tenxcloud.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/tenxcloud.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.zhipin.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/linkedcare.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.xiaobaoonline.com/" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/xiaobao.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.wm-motor.com/" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/weima.png">
        </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td align="center" valign="middle">
        <a href="https://www.iqiyi.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/iqiyi.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.gaiaworks.cn" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/gaiya.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.tydic.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/tianyuandike.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://www.terminus.io" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/terminus.png">
        </a>
      </td>
    </tr>
    <tr>
      <td align="center" valign="middle">
        <a href="https://www.tuya.com" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/tuya.png">
        </a>
      </td>
    </tr>
  </tbody>
</table>
</div>

## 开源生态

MOSN 社区积极拥抱开源生态，与以下开源社区建立了良好的合作关系。

<div>
<table>
  <tbody>
  <tr></tr>
    <tr>
      <td align="center" valign="middle">
        <a href="https://istio.io/" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/istio.png">
        </a>
      </td>
      <td align="center"  valign="middle">
        <a href="https://skywalking.apache.org/" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/skywalking.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://github.com/apache/dubbo-go" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/dubbo-go.png">
        </a>
      </td>
      <td align="center" valign="middle">
        <a href="https://sentinelguard.io/" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/sentinel.png">
        </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td align="center" valign="middle">
        <a href="https://www.sofastack.tech/" target="_blank">
          <img width="222px"  src="https://mosn.io/images/community/sofastack.png">
        </a>
      </td>
      </tr>
    </tbody>
  </table>
</div>

## 社区

请访问 [MOSN 官网](https://mosn.io/zh/docs/community)了解更多关于工作组、Roadmap、社区会议、MOSN 教程等信息。

使用钉钉扫描下面的二维码加入 MOSN 用户交流群。


<p align="center">
<img src="https://github.com/mosn/mosn.io/blob/master/assets/img/dingtalk.jpg?raw=true" width="200">
</p>

## 社区会议

MOSN 社区定期召开社区会议。

- [每双周三晚 8 点（北京时间）](https://ebay.zoom.com.cn/j/96285622161)
- [会议纪要](https://docs.google.com/document/d/12lgyCW-GmlErr_ihvAO7tMmRe87i70bv2xqe4h2LUz4/edit?usp=sharing)

## Landscapes

<p align="center">
<img src="https://landscape.cncf.io/images/left-logo.svg" width="150"/>&nbsp;&nbsp;<img src="https://landscape.cncf.io/images/right-logo.svg" width="200"/>
<br/><br/>
MOSN enriches the <a href="https://landscape.cncf.io/landscape=observability-and-analysis&license=apache-license-2-0">CNCF CLOUD NATIVE Landscape.</a>
</p>
