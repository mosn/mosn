# MOSN Project

[![Build Status](https://travis-ci.org/alipay/sofa-mosn.svg?branch=master)](https://travis-ci.org/alipay/sofa-mosn)
[![codecov](https://codecov.io/gh/alipay/sofa-mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/alipay/sofa-mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/alipay/sofa-mosn)](https://goreportcard.com/report/github.com/alipay/sofa-mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)

MOSN, the short name of Modular Observable Smart Network, is a powerful proxy acting as Service Mesh's data plane like [Envoy](https://www.envoyproxy.io/) but written in GoLang. 
MOSN supports Envoy and Istio's APIs and can be integrated with [Istio](https://istio.io/), so we use MOSN instead of Envoy in [SOFAMesh](https://github.com/alipay/sofa-mesh).
The initial version of MOSN was jointly contributed by Ant Financial and UC Business Unit of Alibaba, and we look forward to the community to participate in the
follow-up development and build an open source excellent project together.

## [MOSN Introduction](docs/Introduction.md)

## Features

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
    
## Quick Start
* [Reference](docs/quickstart/Setup.md)
   
## Docs
* [Catalog](docs/Catalog.md)

## Community
* [Issues](https://github.com/alipay/sofa-mosn/issues)

## Version
* [Changelog](docs/CHANGELOG.md)

## Contribution
+ [How to contribute the code](docs/develop/CONTRIBUTING.md)
+ MOSN is still in its infancy with many capabilities need to be completed, so we welcome everyone to participate in and commit code together.

## Thanks
Thanks to Google, IBM, Lyft for creating the Envoy and Istio system, so that MOSN has a very good reference and we can
quickly land our own ideas.
