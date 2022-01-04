<p align="center">
<img src="https://raw.githubusercontent.com/mosn/community/master/icons/png/mosn-labeled-horizontal.png" width="350" title="MOSN Logo" alt="MOSN logo">
</p>

[![Build Status](https://travis-ci.com/mosn/mosn.svg?branch=master)](https://travis-ci.com/mosn/mosn)
[![codecov](https://codecov.io/gh/mosn/mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/mosn/mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/mosn/mosn)](https://goreportcard.com/report/github.com/mosn/mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

[中文](README_ZH.md)

MOSN (Modular Open Smart Network) is a cloud-native network proxy written in Go language. It is open sourced by Ant Group and verified by hundreds of thousands of production containers in 11.11 global shopping festival. MOSN provides the capabilities of multiple protocol, modularity, intelligent and security. It integrates a large number of cloud-native components, and also integrates a Envoy network library, which is high-performance and easy to expand. MOSN and Istio can be integrated to build Service Mesh, and can also be used as independent L4/L7 load balancers, API gateways, cloud native Ingress, and etc.

## Features

As an open source network proxy, MOSN has the following core functions:

+ Support full dynamic resource configuration through xDS API integrated with Service Mesh.
+ Support proxy with TCP, HTTP, and RPC protocols.
+ Support rich routing features.
+ Support reliable upstream management and load balancing capabilities.
+ Support network and protocol layer observability.
+ Support mTLS and protocols on TLS.
+ Support rich extension mechanism to provide highly customizable expansion capabilities.
+ Support process smooth upgrade.
  
## Download&Install

Use `go get -u mosn.io/mosn`, or you can git clone the repository to `$GOPATH/src/mosn.io/mosn`.

**Notice**

- If you need to use code before 0.8.1, you may needs to run the script `transfer_path.sh` to fix the import path.
- If you are in Linux, you should modify the `SED_CMD` in `transfer_path.sh`, see the comment in the script file.

## Documentation

- [Website](https://mosn.io)
- [Changelog](CHANGELOG.md)

## Contributing

See our [contributor guide](CONTRIBUTING.md).

## Partners

Partners participate in MOSN co-development to make MOSN better.

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

## End Users

The MOSN users. Please [leave a comment here](https://github.com/mosn/community/issues/8) to tell us your scenario to make MOSN better!

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
