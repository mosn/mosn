# Changlog

## v0.8.0

### New features

- Added interface: Support connection returns current availability status
- Manage API to add default help page

### Optimization

- Reduced connections and request default memory allocation
- Optimized machine list information storage in ConfigStore
- Metrics optimization
   - SOFARPC heartbeat requests are no longer recorded in Metrics
   - Optimize shared memory mode for Metrics use
- Optimized profile reading, ignoring empty files and non-json files
- Optimized the xDS client
   - The xDS client is modified to start completely asynchronously without blocking the startup process
   - Optimize xDS client disconnect retry logic

### Bug fix

- Fixed bug where hot upgrade connection migration would fail in TLS Inspector mode
- Fixed a bug where the log rotation configuration could not be updated correctly
- Fixed a bug where the log did not output log time correctly at the Fatal level
- Fixed a bug in which a connected read loop would cause an infinite loop in a particular scenario
- Fixed a bug in HTTP connection count statistics errors
- Fixed a bug that prevented the channel from being properly closed when the connection was closed
- Fixed bug handling of buffers when handling response to BoltV2 protocol
- Fixed concurrency violations when reading and writing persistent configurations
- Fixed concurrency conflicts that receive responses and trigger timeouts

## v0.7.0

### New features

- Added FeatureGates support
- Added Metrics: request processing time in SOFAMosn
- Support for restarting listening sockets that have been closed at runtime

### Refactoring

- Upgraded to Go 1.12.7
- Modified the xDS client startup order, now it will start before the SOFAMosn service

### Bug fix

- Fixed a bug that did not properly trigger a request reset when an RPC request was written.
- Fixed memory leak bugs when no upstream response was received
- Fixed a bug where some of the request parameters would be lost when the HTTP request was retried
- Fixed a bug that could cause panic when DNS resolution failed
- Fixed a bug that didn't time out when the connection was established in TLS Inspector mode
- prometheus output format no longer supports gzip

## v0.6.0

### New features

- Configured the new idle connection timeout `connection_idle_timeout` field, the default value is 90s. SOFAMosn will actively close the connection when the idle connection timeout is triggered
- Error log adds Alert interface and outputs error log with error code format
- Support SDS to obtain TLS certificate

### Refactoring

- Refactored upstream module
   - Refactored the internal Cluster implementation structure
   - Update Host implementation changed from delta update to full update to speed up update
   - Refactored Snapshot implementation
   - Optimized some memory usage
   - Modified the parameters of some interface functions
- Refactored the implementation of Tracing and supports more extensions

### Optimization

- Optimized connected Metrics statistics
- Optimized the output format of Metrics in prometheus mode
- Optimized IO write coroutines to reduce memory footprint

### Bug fix

- Fixed possible concurrency violations when creating Loggers concurrently
- Fixed a bug that caused a panic due to a response and a trigger timeout
- Fixed concurrent bugs when HTTP processing connection reset
- Fixed bug where the log file could not be rotated correctly after it was deleted
- Fixed a concurrent bug when HTTP2 handled goaway

## v0.5.0

### New features

- Configure a hybrid mode that supports both xDS mode and static configuration
- Support management API extensible registration new API
- Supports dynamic update of StreamFilter configuration, which can take effect on all connections

### Refactoring

- Refactored package import path
   - Changed from `github.com/alipay/sofa-mosn` to `sofastack.io/sofa-mosn`

### Optimaztion

- Optimized the output structure of the error log
- Improved implementation of configuration file json parsing
- Optimized memory reuse for large buffer scenarios
- Optimized the use of shared memory for Metrics when first booting

### Bug fix

- Fixed bug where ProxyLogger's log level could not be dynamically updated
- Fixed connection read and write loops that could cause panic bugs
- Fixed a bug that didn't work correctly when deleting multiple clusters at the same time
- Fixed a bug where the number of active requests in Metrics was incorrectly counted in concurrency
- Fixed a bug where the HTTP connection triggered a panic when resetting and receiving a response concurrency

## v0.4.2

### New features

- Support for new profile models
  - Cluster configuration support is set to a separate directory
  - Routing configuration support is set to a separate directory
  - Support for multiple certificate configurations
  - Compatible with old profile models
- Add new Metrics to show basic information
  - version number
  - version of Go used
  - SOFAMosn runtime state
  - the address to listen to
- Support for Metrics filtering
- Support for registering callback functions when SOFAMosn changes state
- Support Request oneway
- Support batch modification of error log level, support batch shutdown of Access log

### Refactoring

- Refactored the Proxy threading model
  - Each request is processed using a separate Goroutine
  - Use the state machine instead of the callback function to request the processing flow to be modified to serial
- Refactored the connection pool selection model, try to avoid selecting the backend to the exception

### Optimization

- Optimized Metrics output performance
- Optimized error log output
- Optimized SOFA RPC protocol parsing performance
- Extended the implementation of the context, reduce the number of nesting layers and optimize performance when compatible with standard contexts

### Bug fix

- Fixed a bug in the parsing part of the json configuration
- Fixed a bug where HTTP would trigger Goroutine leaks in certain scenarios
- Fixed bug in IO write concurrency scenarios that could cause panic
- Fixed a bug where HOST information was not deduplicated

## v0.4.1

### New features

- Metrics added prometheus mode output
- Metrics supports exclusion configuration
- Support dynamic opening and closing of logs, support dynamic adjustment of error log level
- HTTP protocol support 100 continue
- Support Tars protocol
- Connections that support the SOFARPC protocol send heartbeats when idle
- Support for creating connections based on sub-protocols of SOFARPC
- Support for new smooth upgrade methods
- Active health check supports scalable implementation, default is tcp dial
- Memory Multiplexing Module supports scalability
- Load balancing implementation support for scalability
- Profile resolution support is extensible, default is json file parsing

### Refactoring

- Refactored the stream package structure, modified some API interfaces
- Log module modified to asynchronous write log
- Refactored xDS configuration conversion module
- Refactored implementation of the routing chain
- Move some common function functions to the utils package directory

### Optimization

- Route matching optimization, supporting KV matching in specific scenarios
- The request status code recorded in the request message is uniformly converted to the HTTP status code as a standard
- Optimized Tracer implementation to improve the performance of Tracer records
- Optimized performance for profile persistence
- Optimized performance for dynamically updating backend machine lists

### Bug fix

- Fixed deadlock bug in workpool
- Fixed bug with HTTP2 error handling trailer
- Fixed concurrency issues with buffer reuse

## v0.4.0

### New features

- Support gRPC via HTTP2
- Support HTTP/HTTP2 protocol automatic identification
- Tracer supporting SOFA RPC protocol
- Add more routing features
  - Support for retry policy configuration
  - Policy configuration that supports direct response
  - Support for adding and deleting custom fields in HTTP header
  - Support for rewriting the HTTP protocol Host and URI
  - Support for scalable routing implementation
- Support QPS current limiting and rate based current limiting
- Support fault injection
- Support for Mixer
- Support for obtaining MOSN runtime configuration

### Refactoring

- Refactored the protocol framework to support the extension of the SOFA RPC sub-protocol

### Optimization

- Optimized support for the HTTP protocol, performance improvement of about 30%
- Optimized support for HTTP2 protocol, performance improvement of about 100%
- Optimized implementation of TCP Proxy

### Bug fix

- Fixed a bug in smooth upgrades
- Fixed bug with HTTP/HTTP2 protocol processing
- Fixed some bugs with potential memory leaks

## v0.3.0

### New features

- Support for smooth migration of Metrics
- Smooth migration with TLS support

### Optimization

- Optimized CPU and memory usage for SOFARPC protocol resolution

## v0.2.1

### New features

- Support stand-alone TLS switch
- Support Dubbo protocol through XProtocol protocol

## v0.2.0

### New features

- Support for route matching rules with weights
- Added xDS client implementation
  - Support for LDS
  - Support CDS
- Support for four layers of Filter to expand
- Support TLS configuration scalable
- Support for IO processing based on native epoll
- Enhanced scalability for protocol parsing
- Added XProtocol, which can be implemented by XProtocol extension protocol

### Optimization

- Implement memory reuse framework to reduce memory allocation overhead

## v0.1.0

### New features

- Implemented a programmable, scalable network extension framework SOFAMosn
- Achieved the protocol framework
  - Support SOFA RPC protocol
  - Support HTTP protocol
  - Support HTTP2 protocol
- Support for scalable mode based on Stream Filters
- Support back-end cluster management and load balancing
- Support simple route matching rules
- Support smooth restart and smooth upgrade