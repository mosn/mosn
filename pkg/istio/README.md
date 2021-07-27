This package contains some common istio definitions that are not related to the istio version.

An extension for istio version, MUST have:

1. An implementation of `mosn.io/mosn/pkg/istio.XdsStreamConfig` And `mosn.io/mosn/pkg/istio.XdsStreamClient`, and registe the create function by  `mosn.io/mosn/pkg/istio.ResgiterParseAdsConfig`.
2. An implementation of `mosn.io/mosn/pkg/mtls/sds.SdsStreamClient`, qand registe the create function by `mosn.io/mosn/pkg/mtls/sds.RegisterSdsStreamClientFactory`.
3. A main package.
4. A main package in `pkg/networkextension` (MOSN on Envoy).

More istio related functions can be implemented on necessary, for example:

1. some istio related stream filters.
2. cel extensions.
3. global xds info configurations.

In addition to the code, there are other files should be modified:

1. `Makefile`, add a new istio version related action 
2. `etc/script/report.sh` for unit test coverage
