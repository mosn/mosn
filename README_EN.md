# SOFAMosn

[![Build Status](https://travis-ci.org/alipay/sofa-mosn.svg?branch=master)](https://travis-ci.org/alipay/sofa-mosn)
[![codecov](https://codecov.io/gh/alipay/sofa-mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/alipay/sofa-mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/alipay/sofa-mosn)](https://goreportcard.com/report/github.com/alipay/sofa-mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

[中文](README_CN.md)

SOFAMosn is a powerful proxy written in Go that can be used as a Service Mesh's data plane. SOFAMosn is a project in [SOFAStack](https://www.sofastack.tech) and MOSN is the short name of Modular Observable Smart Network. SOFAMosn can be integrated with SOFAMesh through xDS API, also used as a standalone load balancer on layer 4 or 7. In the future, SOFAMosn will support more cloud native scenarios and the core forwarding function of Nginx.

## Features

+ Support full dynamic resource configuration through xDS API integrated with SOFAMesh.
+ Support proxy with TCP, HTTP, and RPC protocols.
+ Support rich routing features.
+ Support reliable upstream management and load balancing capabilities.
+ Support network and protocol layer observability.
+ Support mTLS and protocols on TLS.
+ Support rich extension mechanism to provide highly customizable expansion capabilities.
+ Support process smooth upgrade.
  
## Download&Install

Use `go get -u sofastack.io/sofa-mosn`, or you can git clone the repository to `$GOPATH/src/sofastack.io/sofa-mosn`.

**Notice**

- If you need to use code before 0.5.0, you may needs to run the script ` transfer_path.sh` to fix the import path.
- If you are in Linux, you should modify the `SED_CMD` in `transfer_path.sh`, see the comment in the script file.

## Documentation

See [SOFAMosn docs](https://www.sofastack.tech/projects/sofa-mosn/).

## Contribution

See our [contributor guide](CONTRIBUTING_EN.md).

## Community

Go to [SOFAStack community](https://github.com/sofastack/community) for community specifications and related resources.

Scan the QR code below with [DingTalk](https://www.dingtalk.com) to join the SOFAMosn user group.

![SOFAMosn user group DingTalk QR code](https://gw.alipayobjects.com/mdn/rms_91f3e6/afts/img/A*NyEzRp3Xq28AAAAAAAAAAABkARQnAQ)

## Acknowledgement
SOFAMosn builds on open source projects such as [Envoy](https://github.com/envoyproxy/envoy) and [Istio](https://github.com/istio/istio), thanks to the efforts of the open source community.
