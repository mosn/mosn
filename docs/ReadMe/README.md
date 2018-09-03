# MOSN Project

[![Build Status](https://travis-ci.org/alipay/sofa-mosn.svg?branch=master)](https://travis-ci.org/alipay/sofa-mosn)
[![codecov](https://codecov.io/gh/alipay/sofa-mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/alipay/sofa-mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/alipay/sofa-mosn)](https://goreportcard.com/report/github.com/alipay/sofa-mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

MOSN, the short name of Modular Observable Smart Network, is a powerful proxy acting as Service Mesh's data plane like Envoy but written in golang.
MOSN supports Envoy and Istio's APIs and can be integrated with Istio, and we use MOSN instead of Envoy in SOFAMesh.
The initial version of MOSN was jointly contributed by Ant Financial and UC Business Unit of Alibaba, and we look forward to the community to participate in the
follow-up development and build an open source boutique project together.

## Core Competence

+ Integrated with Istio
    + Integrated with Istio version 1.0 and V4 API and can be run based on full dynamic resource configuration
+ Packet Forwarding
    + Self-contained network server
    + Support TCP proxy
    + Support TProxy mode
+ Multi-protocol support
    + HTTP/1.1，HTTP/2.0
    + SOFARPC
    + Dubbo(ongoing)
+ Routing support
    + Routing in form of virtual host
    + Routing with headers/url/prefix
    + Routing with host metadata in form of subset
    + Routing retry
+ Cluster Management & Load Balance support
    + Connection pool
    + Circuit Breaker
    + Health Checker
    + Random/RR LoadBalance
    + Subset LoadBalance with host's metadata
+ Observable
    + Network layer data
    + Protocol data
+ TLS support
    + HTTP/1.x on TLS
    + HTTP/2 on TLS
    + SOFARPC on TLS
+ Process management
    + Smooth reload
    + Smooth upgrade
+ Scalable ability
    + Support self-defined private protocol
    + Scalable Network/IO ，stream layer

## Feature of Iterative Version
+ [0.2.0](../features/0.2.0/0-2-0-list.md)

## Architecture Design
* [Reference](../design/README.md)

## Quic Start
* [Details](../develop/quickstart.md)
   
## Docs
* [More here](http://www.sofastack.tech/sofa-mesh/docs/mosn-README)

## Community
* [Issues](https://github.com/alipay/sofa-mosn/issues)

## Contribution
+ [How to contribute the code](../develop/CONTRIBUTING.md)
+ MOSN is still in its infancy with many capabilities need to be completed, and many bugs to be fixed.
  So we welcome everyone to participate in and commit code in following but not limited aspect：
   + Completing core routing function
   + Outlier detection
   + Tracing support
   + HTTP/1.x, HTTP/2.0 performance optimization
   + Flow control
   
## Thanks
Thanks to Google, IBM, Lyft for creating the Envoy and Istio system, so that MOSN has a very good reference and we can
quickly land our own ideas.
