![MOSN Logo](https://raw.githubusercontent.com/mosn/community/master/icons/png/mosn-labeled-horizontal.png)

[![Build Status](https://travis-ci.com/sofastack/sofa-mosn.svg?branch=master)](https://travis-ci.com/sofastack/sofa-mosn)
[![codecov](https://codecov.io/gh/alipay/sofa-mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/alipay/sofa-mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/alipay/sofa-mosn)](https://goreportcard.com/report/github.com/alipay/sofa-mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

[中文](README_ZH.md)

MOSN is a network proxy written in Golang. It can be used as a cloud-native network data plane, providing services with the following proxy functions:  multi-protocol, modular, intelligent, and secure. MOSN is the short name of Modular Open Smart Network-proxy. MOSN can be integrated with any Service Mesh wich support xDS API. It can also be used as an independent Layer 4 or Layer 7 load balancer, API Gateway, cloud-native Ingress, etc.

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

Use `go get -u sofastack.io/sofa-mosn`, or you can git clone the repository to `$GOPATH/src/sofastack.io/sofa-mosn`.

**Notice**

- If you need to use code before 0.5.0, you may needs to run the script ` transfer_path.sh` to fix the import path.
- If you are in Linux, you should modify the `SED_CMD` in `transfer_path.sh`, see the comment in the script file.

## Documentation

- [MOSN website](http://mosn.io)
- [Changelog](CHANGELOG.md)

## Contributing

See our [contributor guide](CONTRIBUTING.md).

## Community

Scan the QR code below with [DingTalk](https://www.dingtalk.com) to join the MOSN user group.

![SOFAMosn user group DingTalk QR code](https://gw.alipayobjects.com/mdn/rms_91f3e6/afts/img/A*NyEzRp3Xq28AAAAAAAAAAABkARQnAQ)
