<p align="center">
<img src="https://raw.githubusercontent.com/mosn/community/master/icons/png/mosn-labeled-horizontal.png" width="350" title="MOSN Logo" alt="MOSN logo">
</p>

[![Build Status](https://travis-ci.com/mosn/mosn.svg?branch=master)](https://travis-ci.com/mosn/mosn)
[![codecov](https://codecov.io/gh/mosn/mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/mosn/mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/mosn/mosn)](https://goreportcard.com/report/github.com/mosn/mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

[中文](README_ZH.md)

MOSN (Modular Open Smart Network) is a cloud-native network proxy written in Go language. It is open sourced by Ant Group and verified by hundreds of thousands of production containers in 11.11 global shopping festival. MOSN provides the capabilities of multiple protocol, modularity, intelligent and security. It integrates a large number of cloud-native components, and also integrates a Envoy network library, which is high-performance and easy to expand. MOSN and Istio can be integrated to build Service Mesh, and can also be used as independent L4/L7 load balancers, API gateways, cloud native Ingress, and etc.

## Core capabilities

- Istio integration
   - Integrates Istio 1.10 to run in full dynamic resource configuration mode
- Core forwarding
   - Supports a self-contained server
   - Supports the TCP proxy
   - Supports the UDP proxy
   - Supports transparent traffic hijack mode
- Multi-protocol
   - Supports HTTP/1.1 and HTTP/2
   - Supports protocol extension based on XProtocol framework
   - Supports protocol automatic identification
   - Supports gRPC
- Core routing
   - Supports virtual host-based routing
   - Supports headers/URL/prefix/variable/dsl routing
   - Supports redirect/direct response/traffic mirror routing
   - Supports host metadata-based subset routing
   - Supports weighted routing.
   - Supports retries and timeout configuration
   - Supports request and response headers to add/remove
- Back-end management & load balancing
   - Supports connection pools
   - Supports persistent connection's heart beat handling
   - Supports circuit breaker
   - Supports active back-end health check
   - Supports load balancing policies: random/rr/wrr/edf
   - Supports host metadata-based subset load balancing policies
   - Supports different cluster types: original dst/dns/simple
   - Supports cluster type extension
- Observability
   - Support trace module extension
   - Integrates jaeger/skywalking
   - Support metrics with prometheus style
   - Support configurable access log
   - Support admin API extension
   - Integrates [Holmes](https://github.com/mosn/holmes) to automatic trigger pprof
- TLS
   - Support multiple certificates matches, and TLS inspector mode.
   - Support SDS for certificate get and update
   - Support extensible certificate get, update and verify
   - Support CGo-based cipher suites: SM3/SM4
- Process management
   - Supports hot upgrades
   - Supports graceful shutdown
- Extension capabilities
   - Supports go-plugin based extension
   - Supports process based extension
   - Supports WASM based extension
   - Supports custom extensions configuration
   - Supports custom extensions at the TCP I/O layer and protocol layer
  
## Download&Install

Use `go get -u mosn.io/mosn`, or you can git clone the repository to `$GOPATH/src/mosn.io/mosn`.

## Documentation

- [Website](https://mosn.io)
- [Changelog](CHANGELOG.md)

## Contributing

See our [contributor guide](CONTRIBUTING.md).

## Partners

Partners participate in MOSN co-development to make MOSN better.

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

## End Users

The MOSN users. Please [leave a comment here](https://github.com/mosn/community/issues/8) to tell us your scenario to make MOSN better!

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

## Ecosystem

The MOSN community actively embraces the open source ecosystem and has established good relationships with the following open source communities.

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

## Community

See our community materials on <https://github.com/mosn/community>.

Visit the [MOSN website](https://mosn.io/docs/community/) for more information on working groups, roadmap, community meetings, MOSN tutorials, and more.

Scan the QR code below with [DingTalk(钉钉)](https://www.dingtalk.com) to join the MOSN user group.

<p align="center">
<img src="https://github.com/mosn/mosn.io/blob/master/assets/img/dingtalk.jpg?raw=true" width="200">
</p>

## Community meeting

MOSN community holds regular meetings.

- [Wednesday 8:00 PM CST(Beijing)](https://ebay.zoom.com.cn/j/96285622161) every other week
- [Meeting notes](https://docs.google.com/document/d/12lgyCW-GmlErr_ihvAO7tMmRe87i70bv2xqe4h2LUz4/edit?usp=sharing)

## Landscapes

<p align="center">
<img src="https://landscape.cncf.io/images/left-logo.svg" width="150"/>&nbsp;&nbsp;<img src="https://landscape.cncf.io/images/right-logo.svg" width="200"/>
<br/><br/>
MOSN enriches the <a href="https://landscape.cncf.io/landscape=observability-and-analysis&license=apache-license-2-0">CNCF CLOUD NATIVE Landscape.</a>
</p>
