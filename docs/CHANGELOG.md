# Changelog


## 0.5.0

### New Features

- Support hybrid config model of xDS and static
- Support Admin APIs Extensible
- Support StreamFilters’ dynamic update taking effect on all connections.

### Refatcor

- Refatcor import path
  - change "github.com/alipay/sofa-mosn" to "sofastack.io/sofa-mosn"

### Performance Optimize

- Reorganized the output of the error log
- Improved the implementation of marshal/unmarshal JSON config files
- Optimize memory reuse for large buffer 
- Optimize the processing of shared memory used by Metrics when mosn start

### Bug Fix

- Fix ProxyLogger cannot be updated by logger update APIs
- Fix Read/Write loop panic
- Fix deleting multiple clusters cannot take effect
- Fix computing metrics error when active_requests are  concurrent
- Fix connection panic when http reset stream and receive response are concurrent.

## 0.4.2

### New Features

- Support new config mode
  - Clusters config can be setted as a file path
  - Route config can be setted as a file path
  - Support TLS Contexts configuartion
  - Compatible with version 0.4.1
- Support mosn information metrics
  - mosn version
  - listen address
  - go version
  - mosn state code
- Support metrics sink filter
- Support register callback functions for mosn state changed
- Support request oneway
- Support batch update log level and close accesslog 

### Refatcor

- Refatcored proxy
  - Handle each stream with a goroutine
  - Use states machine instead of callback
- Refatcored connection pool choose logic
  - Try to choose a active connection pool

### Performance Optimize

- Optimize metrics flush 
- Optimize errorlog write
- Optimize sofarpc protocol codec
- Optimize context usage

### Bug Fix

- Fix json marshal bug
- Fix http1 cause goroutines leak bug
- Fix io write casue panic bug
- Fix host info in admin store is not deduplicated bug


## 0.4.1

### Refatcor

- Refatcored stream package, changed some APIs
- Refatcored log package
- Refatcored xds convert
- Refatcored route chain
- Move some common functions into utils package 

### New Features

- Support metrics exclusion
- Support update logger level, enable and disable logger
- Support get route by key-value
- Support add route into virtualhost, clear virtualhost's routes 
- Support http 100 continue
- Support protocol: Tars
- Support sofa rpc heart beat, and create connections by sub protocol

### Performance Optimize

- Optimize config dump
- Optimize tracer implementation
- Optimize host update

### Extension

- Support metrics output extension, default support prometheus and console
- Support load config extension, default is read a json file
- Support health check extension, default is tcp dial
- Support buffer reuse extension
- Support load balancer type extension

### Others

- Add more admin APIs
- RequestInfo records standard http status code 
- New smooth upgrade mode

### Bug Fix

- Fix workpool flush bug
- Fix reset casues dead lock bug
- Fix concurrency bugs
- Fix handle http2 request with trailers bug
- Fix some tiny bugs

## 0.4.0


### Protocol
- Refatcored HTTP1, using MOSN's IO goroutine, 30% performance improvement
- Refatcored HTTP2, using MOSN's IO goroutine, 100% performance improvement
- Support GRPC in HTTP2, support pesudo header query
- Refatcored protocol framework
- Support HTTP1/HTTP2 protocol automatic identification
- Support tracing in SofaRPC

### Traffic Management
- Support retry policy
- Support direct response policy
- Support HTTP header rewrite, rewrite the Host and URI
- Support HTTP custom headers add/delete
- Optimize tcp proxy
- Support rate limit and qps limit
- Support fault inject

### Telemetry
- Support mixer report request/response info

### Extension
- Support more extensions
  - Support a scalable chain routing
  - SofaRPC Protocol support extension

### Others
- Add admin APIs, support update default logger level and get mosn's config
- Use RCU lock to update cluster

### Bug Fix
- Fix some memory leak bugs
- Fix some smooth upgrade bugs
- Fix some HTTP1/HTTP2 bugs
- Fix some tiny bugs


## 0.3.0
- Support router mode 
- Optimize statistic, support smooth upgrade statistic data 
- Support smooth upgrade on TLS
- Optimize cpu usage and memory footprint in SofaRPC
- Fix some bugs

## 0.2.1
- Add TLS disable flag in cluster's host, allows request upstream host in non-TLS mode
- Support dubbo in Xprotocol mode
- Fix some bugs

## 0.2.0
- Support wrr loadbalancer
- Support weighted subset router
- Support listener update/delete, integrated with ISTIO pilot by XDS api
- Support cluster update/delete, integrated with ISTIO pilot by XDS api
- Support network filter extensions, allows config multiple filters
- Support TLS extension, allows customized certificate acquisition
- Support io callback mechanism based on raw epoll/kqueue, optimize support for a large number of connections through io worker pool
- Enhance customized codec extension mechanism in protocol layer
- Add first version of x-protocol extension mechanism
- Add memory reuse framework, use it in io/protocol/stream/protocol layers
- Fix data race cases
- Fix some bugs

## 0.1.0
- Provide usable network programing models & extensible network filter extension framework
- Support protocol framework & sofa rpc implementation (tr/boltv1/boltv2)
- Support stream framework & http2 / sofa rpc stream & client & pool implementation
- Support stream filter extension framework & some filter implementation
- Support proxying http2 / sofa rpc request in a mesh way
- Support cluster management & lb strategies
- Integration with confreg in service discovery model
- Support basic route support
- Support start from a json-format config file
- Support HUP smooth reload
- Support process smooth upgrade
- Process guard by supervisord & log managed by logrotate
