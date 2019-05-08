# MOSN Project

[![Build Status](https://travis-ci.org/alipay/sofa-mosn.svg?branch=master)](https://travis-ci.org/alipay/sofa-mosn)
[![codecov](https://codecov.io/gh/alipay/sofa-mosn/branch/master/graph/badge.svg)](https://codecov.io/gh/alipay/sofa-mosn)
[![Go Report Card](https://goreportcard.com/badge/github.com/alipay/sofa-mosn)](https://goreportcard.com/report/github.com/alipay/sofa-mosn)
![license](https://img.shields.io/badge/license-Apache--2.0-green.svg)


MOSN, the short name of Modular Observable Smart Network, is a powerful proxy acting as Service Mesh's data plane written in GoLang. MOSN integrates with SOFAMesh through XDS API. At the same time, MOSN can be used as a standalone loadbalancer on layer 4 and 7. In the future, MOSN will land in more cloud native scenarios and support the core forwarding functions of nginx.

## Develop States

After our efforts, MOSN’s 0.4.0 version has achieved a lot of improvement on performance and stability, we expect to release 0.5.0 version at the end of May 2019, which will be an stable version used in AntFin’s production environment.
Before that, we will release the latest version named 0.4.x at the end of every month to synchronize the changes from March 2019.

## [MOSN Introduction](docs/Introduction.md)

## Features

+ Support full dynamic resource configuration through XDS api integrated with SOFAMesh.
+ Support proxy with TCP, HTTP, and RPC protocols.
+ Support rich routing features.
+ Support reliable upstream management and load balancing capabilities.
+ Support network and protocol layer observability.
+ Support mTLS and protocols on TLS.
+ Support rich extension mechanism to provide highly customizable expansion capabilities.
+ Support process smooth upgrade.
    
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
