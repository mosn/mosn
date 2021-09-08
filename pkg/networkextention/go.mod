module mosn.io/mosn/pkg/networkextention

go 1.14

require (
	github.com/fsnotify/fsnotify v1.4.10-0.20200417215612-7f4cf4dd2b52
	github.com/hashicorp/go-hclog v0.9.1 // indirect
	github.com/hashicorp/yamux v0.0.0-20181012175058-2f1d1f20f75d // indirect
	mosn.io/api v0.0.0-20210714065837-5b4c2d66e70c
	mosn.io/mosn v0.24.1-0.20210826064226-517b1ec07d2f
	mosn.io/pkg v0.0.0-20210823090748-f639c3a0eb36
)

replace (
	github.com/apache/dubbo-go-hessian2 => github.com/apache/dubbo-go-hessian2 v1.4.1-0.20200516085443-fa6429e4481d // perf: https://github.com/apache/dubbo-go-hessian2/pull/188
	github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.9.4

)
