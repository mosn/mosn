# Changelog

## v1.6.0

### New Features

- PeakEWMA supports configuring activeRequestBias (#2301) @jizhuozhi
- gRPC filter supports UDS (#2309) @wenxuwan
- Support init inherit old mosn config function (#2241) @dengqian
- Allow custom proxy defaultRouteHandlerName (#2308) @fibbery

### Changes

- Example http-sample README adding configuration file link (#2226) @mimani68
- Update wazero to 1.2.1 (#2254) @codefromthecrypt
- Update dependencies (#2230) (#2233) (#2247) (#2302) (#2326) (#2332) (#2333) @dependabot

### Refactoring

- Refactor debug log content, move data into tracef (#2316) @antJack

### Optimization

- Optimize the default EWMA of newly added hosts (#2301) @jizhuozhi
- PeakEwma LB no longer counts error responses (#2323) @jizhuozhi

### Bug fixes

- Fix edfScheduler incorrectly fixing weight to 1 in dynamic load algorithm (#2306) @jizhuozhi
- Fix unstable LB behavior caused by changing the order of cluster hosts (#2258) @dengqian
- Fix panic caused by missing EWMA() method in NilMetrics (#2310) @antJack (#2312) @jizhuozhi
- Fix panic caused by empty cluster hosts when xDS is updated (#2314) @dengqian
- Fix reboot failure due to undeleted UDS socket file when MOSN exits abnormally (#2318) @wenxuwan
- Fix xDS status code not converted error. Fix unhandled istio inbound IPv6 address error (#2144) @kkrrsq
- Fix new connection error when Listener does not close directly during graceful exit of non-hot upgrade (#2234) @hui-cha
- Fix goimports lint error (#2313) @spacewander

## v1.5.0

### New Features

- EdfLoadBalancer supports slow start mode (#2178) @jizhuozhi
- Support cluster exclusive connectionPool (#2281) @yejialiango
- LeastActiveRequest and LeastActiveConnection load balancers support setting active_request_bias (#2286) @jizhuozhi
- Support configuration of metric sampler (#2261) @jizhuozhi
- Add PeakEWMA load balancer (#2253) @jizhuozhi

### Changes

- README update partners & users (#2245) @doujiang24
- Update dependencies (#2242) (#2248) (#2249) @dependabot
- Upgrade the floor Go version supported by MOSN to 1.18 (#2288) @muyuan0

### Optimization

- Use logical clock to make edf scheduler more stable (#2229) @jizhuozhi
- Change log level on missing proxy_on_delete from ERROR to WARN in proxywasm (#2246) @codefromthecrypt
- Receiver names are different for connection object (#2262) @diannaowa
- Disable over strict linters in workflow (#2259) @jizhuozhi
- Disable workflow when PR in draft mode (#2269) @diannaowa
- Use pointer to avoid overhead of duffcopy and duffzero (#2272) @jizhuozhi
- Remove unnecessary imports (#2292) @spacewander
- CI add goimports check (#2297) @spacewander

### Bug fixes

- Fix panic caused by different hosts using the same rander during health check (#2240) @dengqian
- Fix connpool binding conn id (#2263) @antJack 
- Fix saved client stream protocol information to DownStreamProtocol in the context (#2270) @nejisama
- Fix not using the correct Go version for testing (#2288) @muyuan0
- Fix incorrectly assume the variable not found if the real value is '-' (#2174) @3062
- Fix nil interface panic caused by cluster certificate configuration error (#2291) @3062
- Fix parsing error caused by using interface type in the leastActiveRequestLoadBalancer configuration (#2284) @jizhuozhi
- Fix configuration lbConfig not effective (#2299) @3062
- Fix activeRequestBias missing default value and some naming case error (#2298) @jizhuozhi

## v1.4.0

### New Features

- Support record HTTP health check log (#2096) @dengqian
- Add least_connection load balancer (#2184) @dengqian
- Add API to force reconnect and send ADS request (#2183) @dengqian
- Support pprof debug server endpoint configure (#2202) @dengqian
- Integrate with mosn.io/envoy-go-extension, consult [document](https://github.com/mosn/mosn/blob/master/examples/codes/envoy-go-extension/README_EN.md) (#2200) @antJack (#2222) @3062
- Add API to support override registration (mosn/pkg#72) @antJack
- Add a field in variables to record mosn processing time (#2235) @z2z23n0
- Support setting cluster idle_timeout to zero to indicate never timeout (#2197) @antJack

### Refactoring
- Move pprof import to pkg/mosn (#2216) @3062

### Optimization

- Reduce logging for proxywasm benchmarks (#2189) @Crypt Keeper

### Bug fixes

- Enlarge UDP DNS resolve buffer (#2201) @dengqian
- Fix the debug server not inherited during smooth upgrade (#2204) @dengqian
- Fix where new mosn would delete reconfig.sock when smooth upgrade failed (#2205) @dengqian
- Fix HTTP health check URL with query string will be unescaped (#2207) @dengqian
- Fix lb ReqRoundRobin fails to choose host when index exceeds hosts length (#2209) @dengqian
- Fix the problem that after RDS creates a route, the established connection cannot find the route (#2199) @dengqian (#2210) @3062
- Fix old cache values are read after the execution of Variable.Set (mosn/pkg#73) @antJack
- Fix panic caused by DefaultRoller not setting time (mosn/pkg#74) @dengqian
- Advance the timing of metrics initialization to make it work for static config (#2221) @dengqian
- Fix a concurrency problem caused by multiple health checkers sharing rander (#2228) @dengqian
- Set HTTP/1.1 as the HTTP protocol to be sent upstream (#2225) @dengqian
- Completing the missing statistics (#2215) @3062

## v1.3.0

### Refactoring
- Moves to consolidated Proxy-Wasm implementation and enables wazero by default (#2172) [@Crypt Keeper](https://github.com/codefromthecrypt)

### Optimization

- Optimized parsing xDS transparent proxy configuration: add pass-through configuration for unrecognized addresses (#2171) [@3062](https://github.com/3062)
- Optimized the golangci execution flow in CI testing  (#2166) [@taoyuanyuan](https://github.com/taoyuanyuan) (#2167) [@taoyuanyuan](https://github.com/taoyuanyuan)
- Add integrated benchmarks for Proxy-Wasm (#2164) [@Crypt Keeper](https://github.com/codefromthecrypt) (#2169) [@Crypt Keeper](https://github.com/codefromthecrypt)
- Upgrade the minimum version of Go supported by MOSN to 1.17 (#2160) [@Crypt Keeper](https://github.com/codefromthecrypt)
- Fix some problems in the README.md (#2161) [@liaolinrong](https://github.com/liaolinrong)
- Add benchmark (#2173) [@3062](https://github.com/3062)
- subsetLoadBalancer reuse subset entry to optimze alloc/inuse memory (#2119) [@dzdx](https://github.com/dzdx) (#2188) [@liwu](https://github.com/chuailiwu)

### Bug fixes

- Fix a panic problem with connpool_binging when connecting to upstream timeout (#2180) [@EraserTime](https://github.com/EraserTime)
- Fix the problem that retryTime is 0 when cluster LB algorithm is LB_ORIGINAL_DST (#2170) [@3062](https://github.com/3062)
- Fix smooth upgrade failed (#2129) [@Bryce-Huang](https://github.com/Bryce-huang) (#2193) [@3062](https://github.com/3062)
- Modify the way xDS Listener logs are parsed (#2182) [@3062](https://github.com/3062)
- Fix example print error (#2190) [@liaolinrong](https://github.com/liaolinrong)

## v1.2.0

### New Features

- Support for configuring HTTP retry status codes (#2097) [@dengqian](https://github.com/dengqian)
- Add dev container build configuration and instructions (#2108) [@keqingyuan](https://github.com/keqingyuan)
- Support connpool_binding GoAway (#2115) [@EraserTime](https://github.com/EraserTime)
- Support for configuring the listener defaultReadBufferSize (#2133) [@3062](https://github.com/3062)
- Support Proxy-Wasm v2 ABI (#2089) [@lawrshen](https://github.com/lawrshen)
- Support transparent proxy based on iptables tproxy (#2142) [@3062](https://github.com/3062)

### Refactoring

- Remove MOSN's extended context framework and use the variable mechanism instead. Migrate the variable mechanism and memory reuse framework to mosn.io/pkg (#2055) [@nejisama](https://github.com/nejisama)
- Migrating the metrics interface to mosn.io/api (#2124) [@YIDWang](https://github.com/YIDWang)

### Bug fixes

- Fix some missing log parameters (#2141) [@lawrshen](https://github.com/lawrshen)
- Determine if the obtained cookie exists by error (#2136) [@greedying](https://github.com/greedying)

## v1.1.0

### New Features

- TraceLog support for zipkin (#2014) [@fibbery](https://github.com/fibbery)
- Support cloud edge interconnection (#1640) [@CodingSinger](https://github.com/CodingSinger), details can be found in [blog](https://mosn.io/blog/posts/mosn-tunnel/)
- Trace supports plugin extension in the form of Driver, using SkyWalking as trace implementation (#2047) [@YIDWang](https://github.com/YIDWang)
- Support Parsing Extended xDS Stream Filter (#2095) [@Bryce-huang](https://github.com/Bryce-huang)
- stream filter: ipaccess extension implements xDS parsing logic (#2095) [@Bryce-huang](https://github.com/Bryce-huang)
- Add package tar command to MakeFile (#1968) [@doujiang24](https://github.com/doujiang24)

### Changes

- Adjust connection read timeout from buffer.ConnReadTimeout to types.DefaultConnReadTimeout (#2051) [@fibbery](https://github.com/fibbery)
- Fix typo in documentation (#2056) (#2057)[@threestoneliu](https://github.com/threestoneliu) (#2070) [@chenzhiguo](https://github.com/chenzhiguo)
- Update the configuration file of license-checker.yml (#2071) [@kezhenxu94](https://github.com/kezhenxu94)
- New interface for traversing SubsetLB (#2059) (#2061) [@nejisama](https://github.com/nejisama)
- Add SetConfig interface for tls.Conn (#2088) [@antJack](https://github.com/antJack)
- Add Example of xds-server as MOSN control plane (#2075) [@Bryce-huang](https://github.com/Bryce-huang)
- Add error log when HTTP request parsing fails (#2085) [@taoyuanyuan](https://github.com/taoyuanyuan) (#2066) [@fibbery](https://github.com/fibbery)
- Load balancing skips the last selected host on retry  (#2077) [@dengqian](https://github.com/dengqian)
- Access logs support printing traceID, connectionID and UpstreamConnectionID (#2107) [@Bryce-huang](https://github.com/Bryce-huang)

### Refactoring

- Refactor how HostSet is used (#2036) [@dzdx](https://github.com/dzdx)
- Change the connection write data to only support synchronous write mode (#2087) [@taoyuanyuan](https://github.com/taoyuanyuan)

### Optimization

- Optimize the algorithm for creating subset load balancing to reduce memory usage (#2010) [@dzdx](https://github.com/dzdx)
- Support scalable cluster update method operation (#2048) [@nejisama](https://github.com/nejisama)
- Optimize multi-certificate matching logic: match servername first, and match ALPN only after all servernames are unmatched (#2053) [@MengJiapeng](https://github.com/MengJiapeng)

### Bug fixes

- Fix the latest image version in the wasm example to be a fixed version (#2033) [@antJack](https://github.com/antJack)
- Adjust the order of log closing execution when MOSN exits, and fix the problem that some exit logs cannot be output correctly (#2034) [@doujiang24](https://github.com/doujiang24)
- Fix the problem that OriginalDst was not properly processed after matching successfully (#2058) [@threestoneliu](https://github.com/threestoneliu)
- Fix the problem that the protocol conversion scene does not handle exceptions correctly, and add the protocol conversion implementation specification (#2062) [@YIDWang](https://github.com/YIDWang)
- Fix stream proxy not properly handling exception events such as connection write timeout/disconnect (#2080) [@dengqian](https://github.com/dengqian)
- Fix the panic problem that may be caused by the wrong timing of connection event listening (#2082) [@dengqian](https://github.com/dengqian)
- Avoid closing event before event listener connection (#2098) [@dengqian](https://github.com/dengqian)
- HTTP1/HTTP2 protocol save protocol information in context when processing (#2035) [@yidwang](https://github.com/YIDWang)
- Fix possible concurrency issues when pushing xDS (#2101) [@yzj0911](https://github.com/yzj0911)
- If the upstream address variable is not found, it no longer returns null and returns ValidNotFound (#2049) [@songzhibin97](https://github.com/songzhibin97)
- Fix health check does not support xDS (#2084) [@Bryce-huang](https://github.com/Bryce-huang)
- Fix the method of judging the upstream address (#2093) [@dengqian](https://github.com/dengqian)


## v1.0.1

### Changes

- Protocol: Bolt v1 v2 maps status code api.NoHealthUpstreamCode -> ResponseStatusNoProcessor (#2018) [@antJack](https://github.com/antJack).

### Bug fixes

- Should allow AppendGracefulStopStage and AppendBeforeStopStage when MOSN is starting or running (#2029) [@rayowang](https://github.com/rayowang).
- Fix wrong variable in error message when workerPool Panic (#2019) [@antJack](https://github.com/antJack).

## v1.0.0

### Changes

- Protocol: bolt support GoAway (#1993) [@z2z23n0](https://github.com/z2z23n0)
- Protocol: HTTP health check support more configurations (#1999) [@dengqian](https://github.com/dengqian)
- Add new Admin API for query MOSN version (#2002) [@songzhibin97](https://github.com/songzhibin97)
- Exit code change to 2 when mosn start failed in upgrade mode (#2006) [@doujiang24](https://github.com/doujiang24)
- Add a new function to check whether MOSN is in active upgrading state (#2003) [@doujiang24](https://github.com/doujiang24)
- Add new command: stop (#1990) [@Jun10ng](https://github.com/Jun10ng)

### Bug fixes

- Fix the problem that the domain name update result is wrong when there are multiple DNS domain names in StrictDnsCluster (#1994) [@Jun10ng](https://github.com/Jun10ng)
- Fix upgrade state check error when metrics is configured to shared memory (#2011) [@nejisama](https://github.com/nejisama)

## v0.27.0

### New Features

- Support istio v1.10.6 by default. Support switch istio version by make command, current MOSN support v1.10.6 and v1.5.2. (#1910) [@nejisama](https://github.com/nejisama)
- Support use variables to configure route headers add or remove. (#1946) [@MengJiapeng](https://github.com/MengJiapeng)
- Support health check's initialize interval can be configured (#1942) [@rickey17](https://github.com/rickey17)
- Add HTTP Dial for upstream cluster health check (#1942) [@rickey17](https://github.com/rickey17)
- Add extension callback function for tls context created. [@antJack](https://github.com/antJack)
- Support extension for Listener and Connection created. [@antJack](https://github.com/antJack)
- Support graceful shutdown for xprotocol framework. (#1922) [@doujiang24](https://github.com/doujiang24)
- Support graceful shutdown for MOSN (#1922) [@doujiang24](https://github.com/doujiang24)
- Integrated [Holmes](https://github.com/mosn/holmes) for automatically pprof (#1978) [@doujiang24](https://github.com/doujiang24)
- Add Fetch and RequireCert for SDS client [@nejisama](https://github.com/nejisama)
- Add a new tls verify extension: sni verify (#1910) [@nejisama](https://github.com/nejisama)

### Changes

- Upgrade dubbo-go-hessian to v1.10.2 (#1896) [@wongoo](https://github.com/wongoo)
- Add new configuration for upstream cluster: IdleTimeout (#1914) [hui-cha](https://github.com/hui-cha)
- Transfer some upstream cluster configuration constants to `config/v2` package (#1970) [@jizhuozhi](https://github.com/jizhuozhi)
- Add variable to store request raw data in xprotocol protocols implementations [@antJack](https://github.com/antJack)
- Add new configuration for original-dst filter: localhost listener can be used in listener match fallback (#1972) [@nejisama](https://github.com/nejisama)
- Add new configuration for original-dst cluster: use localhost address as the target address [@nejisama](https://github.com/nejisama)
- Do not use vendor mode any more, use go mod instead. (#1997) [@nejisama](https://github.com/nejisama)

### Refactoring

- Refactor MOSN state management and stage management. (#1859) [@doujiang24](https://github.com/doujiang24)
- Shield signal processing extension related interfaces, do not expose semaphore to developers, and modify it to be scalable for behavior after receiving a signal. (#1859) [@doujiang24](https://github.com/doujiang24)
- Use separate IoBuffer for log (#1936) [@nejisama](https://github.com/nejisama)
- Refactor sds provider, support a sds config to generate different tls config (#1958) [@nejisama](https://github.com/nejisama)

### Optimization

- Optimize the irregular module naming in examples. (#1913) [@scaat](https://github.com/scaat)
- Delete some useless fields in connection struct (#1811) [@doujiang24](https://github.com/doujiang24)
- Optimize the heap management in edf loadbalancer (#1920) [@jizhuozhi](https://github.com/jizhuozhi)
- Optimize the error message when get variables returns error (#1952) [@antJack](https://github.com/antJack)
- Optimize the memory reuse in proxy: a finished request received reset will take no effect on memory reuse now [@wangfakang](https://github.com/wangfakang)
- Optimize the memory usage in maglev loadbalancer (#1964) [@baerwang](https://github.com/baerwang)
- Support error log to iobuffer error, support exception handling when log rotation error occurs (#1996) [@nejisama](https://github.com/nejisama)

### Bug fixes

- Fix: when stream id is too large in http2, the connection is not closed. (#1900) [@jayantxie](https://github.com/jayantxie)
- Fix: route error log output is abnormal (#1915) [@scaat](https://github.com/scaat)
- Fix: xprotocol go plugin example build errors (#1899) [@nearmeng](https://github.com/nearmeng)
- Fix: original-dst filter get ip errors being ignored (#1931) [@alpha-baby](https://github.com/alpha-baby)
- Fix: HTTP may cause hangs in high concurrency (#1949) [@alpha-baby](https://github.com/alpha-baby)
- Fix: istio config parse extension spell error (#1927) [LemmyHuang](https://github.com/LemmyHuang)
- Fix: proxy may cause nil panic when get variables (#1953) [@doujiang24](https://github.com/doujiang24)
- Fix: HTTP cannot get reason for connection reset (#1772) [@wangfakang](https://github.com/wangfakang)
- Fix: cannot restart a stopped listener  (#1883) [@lemonlinger](https://github.com/lemonlinger)
- Fix: strict-dns debug log output is abnormal (#1963) [@wangfakang](https://github.com/wangfakang)
- Fix: divide 0 error in edf loadbalancer (#1970) [@jizhuozhi](https://github.com/jizhuozhi)
- Fix: listener may cause nil panic when set deadline (#1981)  [@antJack](https://github.com/antJack)
- Fix: some typo errors [@Jun10ng](https://github.com/Jun10ng) [@fibbery](https://github.com/fibbery)
- Fix: too many goroutines makes go test race failed (#1898) [@alpha-baby](https://github.com/alpha-baby)


## v0.26.0

### Incompatible Change

For implementing new protocols more nature, XProtocol is no longer as a protocol and no subprotocol any more.
XProtocol is a framework to implement protocol easier now.
So, the old existing code for implementing new protocols need some changes,
please see [this doc](reports/xprotocol_0.26.0.md)(In Chinese) for changing the old existing code suit for the new release.

### New Features

- Added the ip_access new filter to manage access control based on IP (#1797). [@Bryce-huang](https://github.com/Bryce-huang)
- Support admin api extends auth functions (#1834). [@nejisama](https://github.com/nejisama)
- The transcode filter module support dynamic phase (#1815). [@YIDWang](https://github.com/YIDWang)
- Added the SetConnectionState method for tls connection in pkg/mtls/crypto/tls.Conn (#1804). [@antJack](https://github.com/antJack)
- Added the after-start and after-stop two new stages, and allow to register handler during these stages. [@doujiang24](https://github.com/doujiang24)
- Support specify the unix domain socket directory by adding the new "uds_dir" configuration (#1829). [@dengqian](https://github.com/dengqian)
- Support choose dynamic protocol convert dynamically and allow register transcoder through go-plugin. [@Tanc010](https://github.com/Tanc010)
- Added more HTTP protocol method to make protocol matcher work properly (#1870). [@XIEZHENGYAO](https://github.com/XIEZHENGYAO)
- Support to set upstream protocol dynamically (#1808). [@YIDWang](https://github.com/YIDWang)
- Support set default HTTP stream config #1886. [@nejisama](https://github.com/nejisama)

### Changes

- Change the default max header size to 8KB (#1837). [@nejisama](https://github.com/nejisama)
- Refactory default HTTP1 and HTTP2 convert, remove the proxy convert, use transcoder filter instead. [@nejisama](https://github.com/nejisama)
- transcoder filter: changed to register trancoder factory instead trancoder (#1879). [@YIDWang](https://github.com/YIDWang)

### Bug fixes

- Fix a HTTP buffer reuse related bug that may leads to nil panics in high concurrency case. [@nejisama](https://github.com/nejisama)
- Fix: get the proper value of variable response_flag, (#1814). [@lemonlinger](https://github.com/lemonlinger)
- Fix: prefix_write not work with "/" (#1826). [@Bryce-huang](https://github.com/Bryce-huang)
- Fix: the reconfig.sock file may be removed unexpectly when killed the old MOSN manually during smoothly upgrade, (#1820). [@XIEZHENGYAO](https://github.com/XIEZHENGYAO)
- Fix the bug in doretry: should not set setupRetry to false directly, since the old response should be skip when the new upstream request has been sent out (#1807). [@taoyuanyuan](https://github.com/taoyuanyuan)
- Should set the inherit config back to the MOSN instance (#1819). [@XIEZHENGYAO](https://github.com/XIEZHENGYAO)
- Should send resetStreamFrame to upstream when cancel grpc context at client side, otherwise server side context won't be done.  [@XIEZHENGYAO](https://github.com/XIEZHENGYAO)
- Should set the resetReason before closing the stream connection, otherwise, may unable to get the real reason (#1828). [@wangfakang](https://github.com/wangfakang)
- Should use the listener that best match when found multi listeners, otherwise, may got 400 error code. [@MengJiapeng](https://github.com/MengJiapeng)
- Fixed panic due to concurrent map iteration and map write during process setting broadcast in HTTP2 protocol. [@XIEZHENGYAO](https://github.com/XIEZHENGYAO)
- Fix memory leak occurred in the binding connpool of XProtocol (#1821). [@Dennis8274](https://github.com/Dennis8274)
- Should close logger at the end, otherwise, may lost log during close MOSN instance (#1845). [@doujiang24](https://github.com/doujiang24)
- Fix panic due to codecClient is nil when got connect timeout event from XProtocol PingPong connection pool (#1849). [@cuiweixie](https://github.com/cuiweixie)
- Health checker not work when the unhealthyThreshold is an empty value (#1853). [@Bryce-huang](https://github.com/Bryce-huang)
- WRR may leads dead recursion in unweightChooseHost #1860. [@alpha-baby](https://github.com/alpha-baby)
- Fix direct response, send hijack should not transcode. [@nejisama](https://github.com/nejisama)
- Fix EDF wrr lb cannot choose a healthy host when there's a unhealthy host with a high weight. [@lemonlinger](https://github.com/lemonlinger)
- Got the wrong CACert filename when converting the listen filter from Istio LDS, MOSN may not listen success (#1893). [@doujiang24](https://github.com/doujiang24)
- The goroutine for resolving hosts in STRICT_DNS_CLUSTER cannot be stopped #1894 [@bincherry](https://github.com/bincherry)

## v0.25.0

### New Features

- Routing configuration supports remove request headers. [@wangfakang](https://github.com/wangfakang)
- Support WASM Reload. [@zu1k](https://github.com/zu1k)
- Integrated SEATA TCC mode, support HTTP protocol. [@dk-lockdown]((https://github.com/dk-lockdown)
- Support boltv2 protocol tracelog. [@nejisama](https://github.com/nejisama)
- New metrics stream filter for gRPC framework.  [@wenxuwan](https://github.com/wenxuwan)
- Support DNS related configuration in xDS cluster config. [@antJack](https://github.com/antJack)

### Refactoring

- Decouple MOSN core and Istio related xDS code. [@nejisama](https://github.com/nejisama)
- Upgrade proxy-wasm-go-host version. [@zhenjunMa](https://github.com/zhenjunMa)
- Refactor networkfilter configuration parse functions, support `AddOrUpdate` and `Get`. [@antJack](https://github.com/antJack)

### Optimization

- Use `mod vendor` instead of `GO111MODULE=off` in Makefile. [@scaat](https://github.com/scaat)
- Move some archived repo code into `mosn.io/pkg`. [@nejisama](https://github.com/nejisama)
- Optimize EDF loadbalancer: random pick host at the first select time. [@alpha-baby](https://github.com/alpha-baby)
- Optimize EDF loadbalancer performance. [@alpha-baby](https://github.com/alpha-baby)
- Optimize boltv2 protocol heartbeat's trigger and reply. [@nejisama](https://github.com/nejisama)
- Optimize HTTP2 retry processing in stream mode, optimize HTTP2 unary request processing in stream mode. [@XIEZHENGYAO](https://github.com/XIEZHENGYAO)
- Ignore CPU numbers limit when use environment variable to set GOMAXPROCS. [@wangfakang](https://github.com/wangfakang)
- Reduce memory alloc when create subset loadbalancer.  [@dzdx]((https://github.com/dzdx)
- Support different listener can independent run same name gRPC Server. [@nejisama](https://github.com/nejisama)

### Bug fixes

- Fix MOSN hangs up when host is empty when retry. [@XIEZHENGYAO](https://github.com/XIEZHENGYAO)
- Fix connections in `msgconnpool` cannot handle connect event. [@RayneHwang](https://github.com/RayneHwang)
- Fix MOSN panic when tracer driver is not inited and someone calls tracer `Enable`. [@nejisama](https://github.com/nejisama)
- Fix boltv2 protocol constructs hijack response version wrong. [@nejisama](https://github.com/nejisama)
- Fix HTTP2 handle connection termination event. [@XIEZHENGYAO](https://github.com/XIEZHENGYAO)
- Fix typo.  [@jxd134](https://github.com/jxd134) [@yannsun](https://github.com/yannsun)
- Fix `ResponseFlag` outputs in `RequestInfo`. [@wangfakang](https://github.com/wangfakang)
- Fix bolt/boltv2 protocol not recalculated the empty data's length. [@hui-cha](https://github.com/hui-cha)

## v0.24.0

### New Features

- Support jaeger to collect OpenTracing message [@Roger](https://github.com/Magiczml)
- Routing configuration new variable configuration mode, you can modify the routing results by modifying the variable [@wangfakang](https://github.com/wangfakang)
- Routing virtualhost matching supports port matching mode [@jiebin](https://github.com/jiebinzhuang)
- Impl envoy filter: [header_to_metadata](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/header_to_metadata_filter) [@antJack](https://github.com/antJack)
- Support graceful for uds [@taoyuanyuan](https://github.com/taoyuanyuan)
- New subset load balancing logic to use the full list of machines for load balancing in scenarios where there is no metadata matching [@nejisama](https://github.com/nejisama)
- MOSN's grpc framework supports graceful stop [@alpha-baby](https://github.com/alpha-baby)

### Optimization

- Optimize health check update mode for Cluster configuration updates [@alpha-baby](https://github.com/alpha-baby)
- Add OnConnectionEvent interface in api.Connection [@CodingSinger](https://github.com/CodingSinger)
- Weighted roundrobin loadbalancer underwriting policy adjusted to normal roundrobin load balancer [@alpha-baby](https://github.com/alpha-baby)
- Enhance interface value in mosn variable model [@antJack](https://github.com/antJack)
- Subset also follows the same underwriting strategy when determining the number of machines and whether they exist [@antJack](https://github.com/antJack)

### Bug fixes

- Dubbo stream filter supports automatic protocol recognition [@Thiswang](https://github.com/Thiswang)
- Fix roundrobin loadbalancer result exception in case of concurrency [@alpha-baby](https://github.com/alpha-baby)
- Fix unix address resolution exception [@taoyuanyuan](https://github.com/taoyuanyuan)
- Fix the exception that HTTP short connection cannot take effect [@taoyuanyuan](https://github.com/taoyuanyuan)
- Fix a memory leak in the TLS over SM3 suite after disconnection [@ZengKe](https://github.com/william-zk)
- HTTP2 support doretry when connection reset by peer or broken pipe [@taoyuanyuan](https://github.com/taoyuanyuan)
- Fix the host information error from the connection pool [@Sharember](https://github.com/Sharember)
- Fix data race bug when choose weighted cluster [@alpha-baby](https://github.com/alpha-baby)
- Return invalid host if host is unhealthy in EdfLoadBalancer [@alpha-baby](https://github.com/alpha-baby)
- Fix the problem that XProtocol routing configuration timeout is invalid [@nejisama](https://github.com/nejisama)

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
