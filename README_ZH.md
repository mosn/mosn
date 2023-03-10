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

## 核心能力

+ Istio 集成
    + 集成 Istio 1.10 版本，可基于全动态资源配置运行
+ 核心转发
    + 自包含的网络服务器
    + 支持 TCP 代理
    + 支持 UDP 代理
    + 支持透明劫持模式
+ 多协议
    + 支持 HTTP/1.1，HTTP/2
    + 支持基于 XProtocol 框架的多协议扩展
    + 支持多协议自动识别
    + 支持 gRPC 协议
+ 核心路由
    + 支持基于 Domain 的 VirtualHost 路由
    + 支持 Headers/Path/Prefix/Variable/DSL 等多种匹配条件的路由
    + 支持重定向、直接响应、流量镜像模式的路由
    + 支持基于 Metadata 的分组路由、支持基于权重的路由
    + 支持基于路由匹配的重试、超时配置
    + 支持基于路由匹配的请求头、响应头处理
+ 后端管理 & 负载均衡
    + 支持连接池管理
    + 支持长连接心跳处理
    + 支持熔断、支持后端主动健康检查
    + 支持 Random/RR/WRR/EDF 等多种负载均衡策略
    + 支持基于 Metadata 的分组负载均衡策略
    + 支持 OriginalDst/DNS/SIMPLE 等多种后端集群模式，支持自定义扩展集群模式
+ 可观察性
    + 支持格式可扩展的 Trace 模块，集成了 jaeger/skywalking 等框架
    + 支持基于 prometheus 的 metrics 格式数据
    + 支持可配置的 AccessLog
    + 支持可扩展的 Admin API
    + 集成 [Holmes](https://github.com/mosn/holmes)，自动监控 pprof
+ TLS
    + 支持多证书匹配模式、支持 TLS Inspector 模式
    + 支持基于 SDS 的动态证书获取、更新机制
    + 支持可扩展的证书获取、更新、校验机制
    + 支持基于 CGo 的国密套件
+ 进程管理
    + 支持平滑升级，包括连接、配置的平滑迁移
    + 支持优雅退出
+ 扩展能力
    + 支持基于 go-plugin 的插件扩展模式的
    + 支持基于进程的扩展模式
    + 支持基于 WASM 的扩展模式
    + 支持自定义扩展配置
    + 支持自定义的四层、七层Filter扩展



## 下载安装

使用 `go get -u mosn.io/mosn` 命令或者将项目代码克隆到 `$GOPATH/src/mosn.io/mosn` 目录中。

## 文档

- [MOSN 官网](https://mosn.io/zh)
- [MOSN 版本更新日志](CHANGELOG_ZH.md)

## 贡献

请参阅[贡献者指南](CONTRIBUTING_ZH.md)。

## 合作伙伴

合作伙伴参与 MOSN 合作开发，使 MOSN 变得更好。

<div class="communnity">
<table>
  <tbody>
  <tr></tr>
    <tr>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.antfin.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/ant.png">
        </a>
      </td>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.aliyun.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/aliyun.png">
        </a>
      </td>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.jd.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/jingdong.png">
        </a>
      </td>
    </tr>
    <tr></tr>
  </tbody>
</table>
</div>

## 终端用户

以下是 MOSN 的用户（部分）：

请在[此处](https://github.com/mosn/community/issues/8)登记并提供反馈来帮助 MOSN 做的更好。

<div>
<table>
  <tbody>
  <tr></tr>
    <tr>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.qunar.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/qunar.jpeg">
        </a>
      </td>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.sf-tech.com.cn/" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/shunfeng.jpeg">
        </a>
      </td>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.58.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/58.png">
        </a>
      </td>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.futuholdings.com/" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/futu.png">
        </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.iqiyi.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/iqiyi.png">
        </a>
      </td>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.zhipin.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/bosszhipin.png">
        </a>
      </td>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.dmall.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/duodian.png">
        </a>
      </td>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.kanzhun.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/kanzhun.png">
        </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.tenxcloud.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/tenxcloud.png">
        </a>
      </td>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.meicai.cn/" trget="_blank">
          <img width="200"  src="https://mosn.io/images/community/meicai.png">
        </a>
      </td>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.xiaobaoonline.com/" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/xiaobao.png">
        </a>
      </td>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.wm-motor.com/" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/weima.png">
        </a>
      </td>
    </tr>
    <tr></tr>
    <tr>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.tuya.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/tuya.png">
        </a>
      </td>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.gaiaworks.cn" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/gaiya.png">
        </a>
      </td>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.tydic.com" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/tianyuandike.png">
        </a>
      </td>
      <td width="222px" align="center" valign="middle">
        <a href="https://www.terminus.io" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/terminus.png">
        </a>
      </td>
    </tr>
  </tbody>
</table>
</div>

## 商业用户

以下是 MOSN 的商业版用户（部分）：

<div class="communnity">
<table>
  <tbody>
  <tr></tr>
    <tr>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.icbc.com.cn/" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/gongshang.png">
        </a>
      </td>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.citicbank.com/" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/zhongxin.png">
        </a>
      </td>
      <td width="222px" align="center"  valign="middle">
        <a href="https://www.hxb.com.cn/" target="_blank">
          <img width="200px"  src="https://mosn.io/images/community/huaxia.png">
        </a>
      </td>
    </tr>
    <tr></tr>
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
