# Changelog

## v0.23.0

### New Features

- Add new networkfilter:grpc. A grpc server can be extended by networkfilter and implemented in the MOSN, reuse MOSN capabilities. [@nejisama](https://github.com/nejisama) [@zhenjunMa](https://github.com/zhenjunMa)
- Add a new extended function for traversal calls in the StreamFilterChain. [@wangfakang](https://github.com/wangfakang)
- Add HTTP 403 status code mapping in the bolt protocol. [@pxzero](https://github.com/pxzero)
- Add the ability to shutdown the upstream connections.  [@nejisama](https://github.com/nejisama)

### Optimization

- Optimize the networkfilter configuration parsed.  [@nejisama](https://github.com/nejisama)
- Support extend proxy config by protocol, optimize the proxy configuration parse timing. [@nejisama](https://github.com/nejisama)
- Add tls conenction's certificate cache, reduce the memory usage. [@nejisama](https://github.com/nejisama)
- Optimize Quick Start Sample. [@nobodyiam](https://github.com/nobodyiam)
- Reuse context when router handler check available. [@alpha-baby](https://github.com/alpha-baby)
- Modify the NewSubsetLoadBalancer's input parameter to interface instead of the determined structure. [@alpha-baby](https://github.com/alpha-baby)
- Add an example of using so plugin to implement a protocol. [@yichouchou](https://github.com/yichouchou)
+ Optimize the method of get environment variable `GOPATH` in the `MAKEFILE`. [@bincherry](https://github.com/bincherry)
+ Support darwin & aarch64 architecture. [@nejisama](https://github.com/nejisama)
- Optimize the logger file flag. [@taoyuanyuan](https://github.com/taoyuanyuan)

### Bug fixes

- Fix the bug of HTTP1 URL encoding. [@morefreeze](https://github.com/morefreeze)
- Fix the bug of HTTP1 URL case-sensitive process. [@GLYASAI](https://github.com/GLYASAI)
- Fix the bug of memory leak in error handling when the tls cipher suite is SM4. [@william-zk](https://github.com/william-zk)

## v0.22.0

### New Features
- Add Wasm extension framework [@antJack](https://github.com/antJack)
- Add x-bolt sub-protocol to allow wasm-based codec for XProtocol [@zonghaishang](https://github.com/zonghaishang)
- Support fallback through SO_ORIGINAL_DST when protocol auto-matching got failed [@antJack](https://github.com/antJack)
- Support Go Plugin mode for XProtocol [@fdingiit](https://github.com/fdingiit)
- Support for network extension [@wangfakang](https://github.com/wangfakang)
- Update to Istio xDS v3 API [@champly](https://github.com/champly)  Branch: [istio-1.7.7](https://github.com/mosn/mosn/tree/istio-1.7.7)

### Optimization

- Remove redundant file path clean when resolving StreamFilter configs [@eliasyaoyc](https://github.com/eliasyaoyc)
- Allow setting a unified callback handler for the StreamFilterChain [@antJack](https://github.com/antJack)
- Support multi-stage execution and remove state lock for the FeatureGate [@nejisama](https://github.com/nejisama)
- Add trace support for HTTP2 [@OrezzerO](https://github.com/OrezzerO)

### Refactoring
- Add StageManger to divide the bootstrap procedure of MOSN into four configurable stages [@nejisama](https://github.com/nejisama)
- Unify the type definitions of XProtocol and move into mosn.io/api package [@fdingiit](https://github.com/fdingiit)
- Add GetTimeout method for XProtocol to replace the variable getter [@nejisama](https://github.com/nejisama)

### Bug fixes
- Fix concurrent read and write for RequestInfo in Proxy [@nejisama](https://github.com/nejisama)
- Fix the safety bug when forwarding the request URI [@antJack](https://github.com/antJack)
- Fix concurrent slice read and write for Router configurations when doing persistence [@nejisama](https://github.com/nejisama)


## v0.21.0

### Optimization

- Upgrade sentinel version to v1.0.2 [@ansiz](https://github.com/ansiz)
- Shrink the read buffer of tls when read timeout, reduce tls memory consumption [@cch123](https://github.com/cch123)
- Add comments and simplify the implementation of the xprotocol protocol connpool [@cch123](https://github.com/cch123)
- Update the mosn registry version [@cadeeper](https://github.com/cadeeper) [@cch123](https://github.com/cch123)

### Refactoring

- Optimize header matching logic when routing, support general RPC routing matching implementation [@nejisama](https://github.com/nejisama)
- Delete some of the original constants and add constants used to describe the mechanism of variables [@nejisama](https://github.com/nejisama)
- Refactor flow control module, support custom callback extension, realize the ability to customize filter conditions and modify context information, etc [@ansiz](https://github.com/ansiz)

### Bug fixes

- Fix metrics statistics error when request is abnormal [@cch123](https://github.com/cch123)
- Fix the bug that the URL is not escaping before forwarding HTTP request [@antJack](https://github.com/antJack)
- Fix the variable injection errors in HTTP protocol, Fix the bug that routing rewrite is not supported in the HTTP2 protocol [@nejisama](https://github.com/nejisama)

### New Features

- Support Domain-Specific Language route implementation [@CodingSinger](https://github.com/CodingSinger)
- StreamFilter supports the dynamic link libraries written in Go [@CodingSinger](https://github.com/CodingSinger)
- VirtualHost supports per_filter_config configuration in routing configuration [@machine3](https://github.com/machine3)
- Xprotocol supports dubbo thrift protocol [@cadeeper](https://github.com/cadeeper)

## v0.20.0

### Optimization

- Add UDS address prefix check before UDS resolution when TCP address resolution fails [@wangfakang](https://github.com/wangfakang)
- Optimized the retrial interval for connection pool acquisition [@nejisama](https://github.com/nejisama)
- Add global switch for write loop mode [@nejisama](https://github.com/nejisama)
- Optimize auto protocol matching and add test cases [@taoyuanyuan](https://github.com/taoyuanyuan)
- Replace the headers with more efficient variables [@CodingSinger](https://github.com/CodingSinger)
- Pool the writeBufferChan timer to reduce overhead [@cch123](https://github.com/cch123)
- Add MOSN failure detail info into TraceLog [@nejisama](https://github.com/nejisama)
- New read done channel in HTTP protocol processing [@alpha-baby](https://github.com/alpha-baby)
- Enhance logger rotator [@nejisama](https://github.com/nejisama)

### Refactoring

- Upgrade to golang 1.14.13 [@nejisama](https://github.com/nejisama)
- Refactor router chain extension mode to the route handler extension mode, support different router handler configuration [@nejisama](https://github.com/nejisama)
- Refactor MOSN extended configuration, support load config according to configuration order [@nejisama](https://github.com/nejisama)

### Bug fixes

- Fix the bug no provider available occurred after dubbo2.7.3 [@cadeeper](https://github.com/cadeeper)
- Fix the bug that UDS connections were treated as TCP connections in netpoll mode [@wangfakang](https://github.com/wangfakang)
- Fix the problem that the HTTP Header cannot be obtained correctly when it is set to an empty value [@ianwoolf](https://github.com/ianwoolf)

### New Features

- Support old Mosn transfer configuration to new Mosn through UDS to solve the issue that Mosn in XDS mode cannot be smoothly upgraded [@alpha-baby](https://github.com/alpha-baby)
- Automatic protocol identification supports the identification of XProtocol [@cadeeper](https://github.com/cadeeper)
- Support configuration of the keepalive parameters for XProtocol [@cch123](https://github.com/cch123)
- Support more detailed time tracking [@nejisama](https://github.com/nejisama)
- Support metrics lazy registration to optimize metrics memory when number of service in cluster is too large [@champly](https://github.com/champly)
- Add setter function for default XProtocol multiplex connection pool size [@cch123](https://github.com/cch123)
- Support netpoll [@cch123](https://github.com/cch123)
- Support broadcast [@dengqian](https://github.com/dengqian)
- Support get tls configurations from LDS response [@wZH-CN](https://github.com/wZH-CN)
- Add ACK response for SDS [@wZH-CN](https://github.com/wZH-CN)

## v0.19.0

### Optimization

- Use the latest TLS memory optimization scheme [@cch123](https://github.com/cch123)
- Proxy log optimization to reduce memory escape [@taoyuanyuan](https://github.com/taoyuanyuan)
- Increase the maximum number of connections limit [@champly](https://github.com/champly)
- When AccessLog fails to obtain variables, use "-" instead [@champly](https://github.com/champly)
- MaxProcs supports configuring automatic recognition based on CPU usage limits [@champly](https://github.com/champly)
- Allow specifying network for cluster [@champly](https://github.com/champly)

### Refactoring

- Refactored the StreamFilter framework. The network filter can reuse the stream filter framework [@antJack](https://github.com/antJack)

### Bug fixes

- Fix HTTP Trace get URL error [@wzshiming](https://github.com/wzshiming)
- Fix the ConnectTimeout parameter of xDS cluster is not converted [@dengqian](https://github.com/dengqian)
- Fix the upstreamHostGetter method gets the wrong hostname [@dengqian](https://github.com/dengqian)
- Fix tcp proxy close the connection abnormally [@dengqian](https://github.com/dengqian)
- Fix the lack of default configuration of mixer filter, resulting in a nil pointer reference [@glyasai](https://github.com/glyasai)
- Fix HTTP2 direct response not setting `Content-length` correctly [@wangfakang](https://github.com/wangfakang)
- Fix the nil pointer reference in getAPISourceEndpoint [@dylandee](https://github.com/dylandee)
- Fix memory increase caused by too many Timer applications when Write is piled up [@champly](https://github.com/champly)
- Fix the problem of missing stats when Dubbo Filter receives an illegal response [@champly](https://github.com/champly)

## v0.18.0

### New Features

- Add MOSN configure extension [@nejisama](https://github.com/nejisama)
- Add MOSN configuration tool [mosn/configure](https://github.com/mosn/configure), improve user configure experience [@cch123](https://github.com/cch123)

### Optimization

- Avoid copying http response body [@wangfakang](https://github.com/wangfakang)
- Upgrade `github.com/TarsCloud/TarsGo` package, to v1.1.4 [@champly](https://github.com/champly)
- Add test for various connpool [@cch123](https://github.com/cch123)
- Use sync.Pool to reduce memory cost by TLS connection outBuf [@cch123](https://github.com/cch123)
- Reduce xprotocol lock area [@cch123](https://github.com/cch123)
- Remove useless parameter of `network.NewClientConnection` method, remove ALPN detection in `Dispatch` method of struct `streamConn` [@nejisama](https://github.com/nejisama)
- Add `TerminateStream` API to `StreamReceiverFilterHandler`, with which stream can be reset during handling [@nejisama](https://github.com/nejisama)
- Add client TLS fallback [@nejisama](https://github.com/nejisama)
- Fix TLS HashValue in host [@nejisama](https://github.com/nejisama)
- Fix disable_log admin api typo [@nejisama](https://github.com/nejisama)

### Bug fixes

- Fix `go mod tidy` failing [@champly](https://github.com/champly)
- Fix `ResourceExhausted: grpc: received message larger than max` when MOSN receive > 4M XDS messages [@champly](https://github.com/champly) 
- Fix fault tolerance unit-test [@wangfakang](https://github.com/wangfakang)
- Fix MOSN reconfig fails when `MOSNConfig.servers[].listeners[].bind_port` is `false` [@alpha-baby](https://github.com/alpha-baby)
- Set timeout for local write buffer send, avoid goroutine leak [@cch123](https://github.com/cch123)
- Fix deadloop when TLS timeout [@nejisama](https://github.com/nejisama)
- Fix data isn't modified by `SetData` method in `dubbo.Frame` struct [@lxd5866](https://github.com/lxd5866)

## v0.17.0

### New Features

- Add header max size configuration option. [@wangfakang](https://github.com/wangfakang)
- Add protocol impement choice whether need workerpool mode. And support workerpool mode concurrent configuration. [@cch123](https://github.com/cch123)
- Add UDS feature for listener. [@CodingSinger](https://github.com/CodingSinger)
- Add dubbo protocol use xDS httproute config filter. [@champly](https://github.com/champly)

### Optimization

- Optimiza http situation buffer malloc. [@wangfakang](https://github.com/wangfakang)
- Optimize RWMutex for SDS StreamClient. [@nejisama](https://github.com/nejisama)
- Update hessian2 v1.7.0 lib. [@cch123](https://github.com/cch123)
- Modify NewStream interface, use callback replace direct. [@cch123](https://github.com/cch123)
- Refactor XProtocol connect pool, support pingpong mode, mutiplex mode and bind mode. [@cch123](https://github.com/cch123)
- Optimize XProtocol mutiplex mode, support Host max connect configuration. [@cch123](https://github.com/cch123)
- Optimize route regex config avoid dump unuse config. [@wangfakang](https://github.com/wangfakang)

### Bug fixes

- Fix README ant logo invalid address. [@wangfakang](https://github.com/wangfakang)
- Fix header override content when set a longer header to request header. [@cch123](https://github.com/cch123)
- Fix Dubbo protocol analysis attachment maybe panic. [@champly](https://github.com/champly)

## v0.16.0

### Optimization

- Logger Roller supports the custom Roller. [@wenxuwan](https://github.com/wenxuwan)
- Add a SendHijackReplyWithBody API for streamFilter. [@wenxuwan](https://github.com/wenxuwan)
- The configuration adds option of turning off the smooth upgrade. If the smooth upgrade is turned off, different instances of MOSN can be started on the same machine. [@cch123](https://github.com/cch123)
- Optimize the MOSN integration test framework and add more unit test cases. [@nejisama](https://github.com/nejisama) [@wangfakang](https://github.com/wangfakang) [@taoyuanyuan](https://github.com/taoyuanyuan)
- DirectResponse route configuration supports the update mode of XDS. [@wangfakang](https://github.com/wangfakang)
- Add a new field of TLSContext for clusterManager configuration. [@nejisama](https://github.com/nejisama)

### Bug fixes

- Fix the bug that UDP connection timeout during the smooth upgrade will cause an endless loop. [@dengqian](https://github.com/dengqian)
- Fix the bug that call DirectResponse in the SendFilter will cause an endless loop. [@taoyuanyuan](https://github.com/taoyuanyuan)
- Fix concurrency conflicts in HTTP2 stream counting. [@wenxuwan](https://github.com/wenxuwan)
- Fix the bug that UDP connection read timeout cause data loss. [@dengqian](https://github.com/dengqian)
- Fix the bug that the response StatusCode cannot be recorded correctly due to the loss of the protocol flag when doing a retry. [@dengqian](https://github.com/dengqian)
- Fix the protocol boltv2 decode error. [@nejisama](https://github.com/nejisama)
- Fix the bug that listener cannot be restarted automatically when listener panic. [@alpha-baby](https://github.com/alpha-baby)
- Fix the bug that NoCache flag is invalid in variable. [@wangfakang](https://github.com/wangfakang)
- Fix concurrency conflicts in SDS reconnect. [@nejisama](https://github.com/nejisama)

## v0.15.0

### New Features

- Routing Path Rewrite supports configuring the content of Rewrite by regular expression [@liangyuanpeng](https://github.com/liangyuanpeng)
- Configure new fields: Extended configuration fields, you can start the configuration by extending the configuration fields; Dubbo service discovery configuration via extended configuration fields [@cch123](https://github.com/cch123)
- New DSL feature for easy control of request processing behavior [@wangfakang](https://github.com/wangfakang)
- Extended implementation of StreamFilter with new traffic mirroring function [@champly](https://github.com/champly)
- Listener configuration adds UDP support [@dengqian](https://github.com/dengqian)
- Configuration format support YAML format parsing [@GLYASAI](https://github.com/GLYASAI)
- Routing support for HTTP redirect configuration [@knight42](https://github.com/knight42)

### Optimization

- Istio's stats filter for personalizing metrics based on matching criteria [@wzshiming](https://github.com/wzshiming)
- Metrics configuration support to configure the output percentage of the Histogram [@champly](https://github.com/champly)
- StreamFilter New state for aborting requests directly and not responding to clients [@taoyuanyuan](https://github.com/taoyuanyuan)
- XProtocol hijack response support carry body [@champly](https://github.com/champly)
- Apache SkyWalking upgrade to version 0.5.0 [arugal](https://github.com/arugal)
- Upstream Connection TLS State Determination Modification to support the determination of whether a connection needs to be re-established via a TLS-configured Hash [@nejisama](https://github.com/nejisama)
- Optimize DNS cache logic to prevent DNS flooding issues that can be caused when DNS fails [@wangfakang](https://github.com/wangfakang)

### Bug fixes

- Fix the bug that XProtocol protocols determine protocol errors in scenarios with multiple protocols when TLS encryption is enabled [@nejisama](https://github.com/nejisama)
- Fix bug in AccessLog where variables of prefix match type don't work [@dengqian](https://github.com/dengqian)
- Fix bug where Listener configuration parsing is not handled correctly [@nejisama](https://github.com/nejisama)
- Fix Router/Cluster bug that fails to save when the Name field contains a path separator in the file persistence configuration type [@nejisama](https://github.com/nejisama)

## v0.14.0

### New Features

- Support for Istio 1.5.X [@wangfakang](https://github.com/wangfakang) [@trainyao](https://github.com/trainyao) [@champly](https://github.com/champly)
  - go-control-plane upgrade to version 0.9.4
  - xDS support for ACK, new Metrics for xDS.
  - Istio sourceLabels filtering support
  - probe interface with pilot-agent support
  - support for more startup parameters, adapting to Istio agent startup scenarios
  - gzip, strict-dns, original-dst support for xDS updates.
  - Remove Xproxy Logic
- Maglev Load Balancing Algorithm Support [@trainyao](https://github.com/trainyao)
- New connection pool implementation for supporting message class requests [@cch123](https://github.com/cch123)
- New Metrics for TLS Connection Switching [@nejisama](https://github.com/nejisama)
- Metrics for adding HTTP StatusCode [@dengqian](https://github.com/dengqian)
- Add Metrics Admin API output [@dengqian](https://github.com/dengqian)
- New interface to query the number of current requests for proxy [@zonghaishang](https://github.com/zonghaishang)
- Support for HostRewrite Header [@liangyuanpeng](https://github.com/liangyuanpeng)

### Optimization

- Upgrade tars dependencies to fix compilation issues with higher versions of Golang [@wangfakang](https://github.com/wangfakang)
- xDS Configuration Analysis Upgrade Adaptation Istio 1.5.x [@wangfakang](https://github.com/wangfakang)
- Optimize log output from proxy [@wenxuwan](https://github.com/wenxuwan)
- DNS Cache default time changed to 15s [@wangfakang](https://github.com/wangfakang)
- HTTP Parameter Route Matching Optimization [@wangfakang](https://github.com/wangfakang)
- Upgrade the fasthttp library [@wangfakang](https://github.com/wangfakang)
- Optimizing Dubbo Request Forwarding Encoding [@zonghaishang](https://github.com/zonghaishang)
- Request max body configurable for HTTP support [@wangfakang](https://github.com/wangfakang)

### Bug fixes

- Fix Dubbo Decode bug that fails to parse attachment [@champly](https://github.com/champly)
- Fix bug where streams could be created before HTTP2 connection is established [@dunjut](https://github.com/dunjut)
- Fix HTTP2 Handling Trailer null pointer exception [@taoyuanyuan ](https://github.com/taoyuanyuan)
- Fix bug where HTTP request headers are not standardized by default [@nejisama](https://github.com/nejisama)
- Fix panic exceptions caused by disconnected connections during HTTP request processing [@wangfakang](https://github.com/wangfakang)
- Fix read/write lock copy issue with dubbo registry [@champly](https://github.com/champly)

## v0.13.0

### New Features

- Support Strict DNS Cluster [@dengqian](https://github.com/dengqian)
- Stream Filter [@wangfakang](https://github.com/wangfakang) that supports GZip processing
- Dubbo service registry complete Beta version [@cch123](https://github.com/cch123)
- Stream Filter [@NeGnail](https://github.com/NeGnail) that supports standalone fault isolation
- Integrated Sentinel flow limiting capability [@ansiz](https://github.com/ansiz)

### Optimization

- Optimize implementation of EDF LB and re-implement WRR LB using EDF [@CodingSinger](https://github.com/CodingSinger)
- Configure to get ADMIN API optimizations, add features and environment variable related ADMIN API [@nejisama](https://github.com/nejisama)
- Update that triggers a health check when updating Host changed from asynchronous mode to synchronous mode [@nejisama](https://github.com/nejisama)
- Updated the Dubbo library to optimize the performance of Dubbo Decode [@zonghaishang](https://github.com/zonghaishang)
- Optimize Metrics output in Prometheus, using regularity to filter out illegal Key [@nejisama](https://github.com/nejisama)
- Optimize MOSN's return status code [@wangfakang](https://github.com/wangfakang)

### Bug fix

- Fix concurrent conflict issues with health check registration callback functions [@nejisama](https://github.com/nejisama)
- Fix the error where the configuration persistence function did not handle the null configuration correctly [@nejisama](https://github.com/nejisama)
- Fix the problem that DUMP as a file fails when ClusterName/RouterName is too long [@nejisama](https://github.com/nejisama)
- Fix the problem of not getting the XProtocol protocol correctly when getting it [@wangfakang](https://github.com/wangfakang)
- Fix the problem with fetching the wrong context when creating StreamFilter [@wangfakang](https://github.com/wangfakang)

## v0.12.0

### New Features

- Support Skywalking [@arugal](https://github.com/arugal)
- Stream Filter adds a new phase of Receive Filter execution, which allows you to execute Receive Filter [@wangfakang](https://github.com/wangfakang) again after MOSN has finished routing Host
- HTTP2 supports streaming [@peacocktrain](https://github.com/peacocktrain) [@taoyuanyuan](https://github.com/taoyuanyuan)
- FeatureGate adds interface KnownFeatures to output current FeatureGate status [@nejisama](https://github.com/nejisama)
- Provide a protocol-transparent way to obtain requested resources (PATH, URI, ARG), with the definition of resources defined by each protocol itself [@wangfakang](https://github.com/wangfakang)
- New load balancing algorithm
  - Support for ActiveRequest LB [@CodingSinger](https://github.com/CodingSinger)
  - Support WRR LB [@nejisama](https://github.com/nejisama)

### Optimize

- XProtocol protocol engine optimization [@neverhook](https://github.com/neverhook)
  - Modifies the XProtocol heartbeat response interface to support the protocol's heartbeat response to return more information
  - Optimize connpool for heartbeat triggering, only heartbeats will be triggered if the protocol for heartbeats is implemented
- Dubbo library dependency version updated from v1.5.0-rc1 to v1.5.0 [@cch123](https://github.com/cch123)
- API Adjustments, HostInfo added health check related interface [@wangfakang](https://github.com/wangfakang)
- Optimize circuit breaking function [@wangfakang](https://github.com/wangfakang)
- Responsible for balanced selection logic simplification, Hosts of the same address multiplex the same health check mark [@nejisama](https://github.com/nejisama) [@cch123](https://github.com/cch123)
- Optimize HTTP building logic and improve HTTP building performance [@wangfakang](https://github.com/wangfakang)
- Log rotation logic triggered from writing logs, adjusted to timed trigger [@nejisama](https://github.com/nejisama)
- Typo fixes [@xujianhai666](https://github.com/xujianhai666) [@candyleer](https://github.com/candyleer)

### Bug Fix

- Fix the xDS parsing fault injection configuration error [@champly](https://github.com/champly)
- Fix the request hold issue caused by the MOSN HTTP HEAD method [@wangfakang](https://github.com/wangfakang)
- Fix a problem with missing StatusCode mappings in the XProtocol engine [@neverhook](https://github.com/neverhook)
- Fix the bug for DirectReponse triggering retries [@taoyuanyuan](https://github.com/taoyuanyuan)

## v0.11.0

### New features

- Support the extension of Listener Filter, the transparent hijacking capability is implemented based on Listener Filter [@wangfakang](https://github.com/wangfakang)
- New Set method for variable mechanism [@pxzero](https://github.com/pxzero)
- Added automatic retry and exception handling when SDS Client fails [@taoyuanyuan](https://github.com/taoyuanyuan)
- Improve TraceLog and support injecting context [@nejisama](https://github.com/nejisama)

### Refactoring

- Refactored XProtocol Engine and reimplemented SOFARPC protocol [@neverhook](https://github.com/neverhook)
  - Removed SOFARPC Healthcheck filter and changed to xprotocol's built-in heartbeat implementation
  - Removed the original protocol conversion (protocol conv) support of the SOFARPC protocol, and added a new protocol conversion extension implementation capability based on stream filter
  - xprotocol adds idle free and keepalive
  - Protocol analysis and optimization
- Modify the Encode method parameters of HTTP2 protocol [@taoyuanyuan](https://github.com/taoyuanyuan)
- Streamlined LDS interface parameters [@nejisama](https://github.com/nejisama)
- Modified the routing configuration model, abandoned `connection_manager` [@nejisama](https://github.com/nejisama)

### Optimization

- Optimize Upstream dynamic domain name resolution mechanism [@wangfakang](https://github.com/wangfakang)
- Optimized TLS encapsulation, added error log, and modified timeout period in compatibility mode [@nejisama](https://github.com/nejisama)
- Optimize timeout setting, use variable mechanism to set timeout [@neverhook](https://github.com/neverhook)
- Dubbo parsing library dependency upgraded to 1.5.0 [@cch123](https://github.com/cch123)
- Reference path migration script adds OS adaptation [@taomaree](https://github.com/taomaree)

### Bug fix

- Fix the problem of losing query string during HTTP2 protocol forwarding [@champly](https://github.com/champly)

## v0.10.0

### New features

- Support multi-process plugin mode
- Startup parameters support service-meta parameters
- Supports abstract uds mode to mount sds socket

### Refactoring

- Separate some mosn base library code into mosn.io/pkg package (github.com/mosn/pkg)
- Separate mosn interface definition to mosn.io/api package (github.com/mosn/api)

### Optimization

- The log basic module is separated into mosn.io/pkg, and the log of mosn is optimized
- Optimize FeatureGate
- Added processing when failed to get SDS configuration
- When CDS deletes a cluster dynamically, it will stop the health check corresponding to the cluster
- The callback function when sds triggers certificate update adds certificate configuration as a parameter

### Bug fixes

- Fixed a memory leak issue when SOFARPC Oneway request failed
- Fixed the issue of 502 error when receiving non-standard HTTP response
- Fixed possible conflicts during DUMP configuration
- Fixed the error of Request and Response Size of TraceLog statistics
- Fixed write timeout failure due to concurrent write connections
- Fixed serialize bug
- Fixed the problem that the memory reuse buffer is too large when the connection is read, causing the memory consumption to be too high
- Optimize Dubbo related implementation in XProtocol

### v0.9.0

### New features

- Support variable mechanism, accesslog is modified to use variable mechanism to obtain information

## Refactoring

- Refactored package reference path for `sofastack.io/sofa-mosn` to `mosn.io/mosn`

### Bug fix

- Fixed the bug that buf is not empty when Write is connected
- Fixed HTTP2 stream counting bug
- Fix memory leak caused by proxy coroutine panic
- Fix memory leak caused by read and write coroutine stuck in specific scenarios
- Fix the bug of xDS concurrent processing
- `make image` output image modification, modified to a MOSN example
- Fixed the field of getting CallerAPP in TraceLog of SOFA RPC

## v0.8.1

### New features

- Metrics added: count of requests that MOSN failed to process

### Optimization

- Improved the write performance of Metrics shared memory with MMAP
- Reduced the default coroutine pool size and optimize memory usage
- Optimized log output

### Bug fix

- Fixed a bug where if there is a log file when MOSN starts, it is not rotated normally.

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
- Added Metrics: request processing time in MOSN
- Support for restarting listening sockets that have been closed at runtime

### Refactoring

- Upgraded to Go 1.12.7
- Modified the xDS client startup order, now it will start before the MOSN service

### Bug fix

- Fixed a bug that did not properly trigger a request reset when an RPC request was written.
- Fixed memory leak bugs when no upstream response was received
- Fixed a bug where some of the request parameters would be lost when the HTTP request was retried
- Fixed a bug that could cause panic when DNS resolution failed
- Fixed a bug that didn't time out when the connection was established in TLS Inspector mode
- prometheus output format no longer supports gzip

## v0.6.0

### New features

- Configured the new idle connection timeout `connection_idle_timeout` field, the default value is 90s. MOSN will actively close the connection when the idle connection timeout is triggered
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
  - MOSN runtime state
  - the address to listen to
- Support for Metrics filtering
- Support for registering callback functions when MOSN changes state
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

- Implemented a programmable, scalable network extension framework MOSN
- Achieved the protocol framework
  - Support SOFA RPC protocol
  - Support HTTP protocol
  - Support HTTP2 protocol
- Support for scalable mode based on Stream Filters
- Support back-end cluster management and load balancing
- Support simple route matching rules
- Support smooth restart and smooth upgrade
