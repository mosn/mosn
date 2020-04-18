<p align="center">
<img src="https://raw.githubusercontent.com/mosn/community/master/icons/png/mosn-labeled-horizontal.png" width="350" title="MOSN Logo" alt="MOSN logo">
</p>

[![Build Status](https://travis-ci.com/mosn/mosn.svg?branch=master)](https://travis-ci.com/mosn/mosn)
[![codecov](https://codecov.io/gh/mosn/mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/mosn/mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/mosn/mosn)](https://goreportcard.com/report/github.com/mosn/mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

[中文](README_ZH.md)

MOSN is a network proxy written in Golang. It can be used as a cloud-native network data plane, providing services with the following proxy functions:  multi-protocol, modular, intelligent, and secure. MOSN is the short name of Modular Open Smart Network-proxy. MOSN can be integrated with any Service Mesh which support xDS API. It also can be used as an independent Layer 4 or Layer 7 load balancer, API Gateway, cloud-native Ingress, etc.

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

## Community

See our community materials on <https://github.com/mosn/community>.

Scan the QR code below with [DingTalk(钉钉)](https://www.dingtalk.com) to join the MOSN user group.

<p align="center">
<img src="https://gw.alipayobjects.com/mdn/rms_91f3e6/afts/img/A*NyEzRp3Xq28AAAAAAAAAAABkARQnAQ" width="150">
</p>


## Landscapes

<p align="center">
<img src="https://landscape.cncf.io/images/left-logo.svg" width="150"/>&nbsp;&nbsp;<img src="https://landscape.cncf.io/images/right-logo.svg" width="200"/>
<br/><br/>
MOSN enriches the <a href="https://landscape.cncf.io/landscape=observability-and-analysis&license=apache-license-2-0">CNCF CLOUD NATIVE Landscape.</a>
</p>

