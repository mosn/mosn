module mosn.io/mosn

go 1.12

require (
	bou.ke/monkey v1.0.2
	github.com/SkyAPM/go2sky v0.5.0
	github.com/TarsCloud/TarsGo v1.1.4
	github.com/alibaba/sentinel-golang v1.0.2-0.20210112133552-db6063eb263e
	github.com/apache/dubbo-go-hessian2 v1.7.0
	github.com/apache/thrift v0.13.0
	github.com/c2h5oh/datasize v0.0.0-20171227191756-4eba002a5eae
	github.com/cncf/udpa/go v0.0.0-20191209042840-269d4d468f6f
	github.com/dchest/siphash v1.2.1
	github.com/dgrijalva/jwt-go v3.2.0+incompatible
	github.com/envoyproxy/go-control-plane v0.9.4
	github.com/ghodss/yaml v1.0.0
	github.com/go-chi/chi v4.1.0+incompatible
	github.com/gogo/protobuf v1.3.1
	github.com/golang/mock v1.4.4
	github.com/golang/protobuf v1.3.5
	github.com/google/cel-go v0.5.1
	github.com/hashicorp/go-plugin v1.0.1
	github.com/json-iterator/go v1.1.9
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/testing v0.0.0-20191001232224-ce9dec17d28b // indirect
	github.com/lestrrat/go-jwx v0.0.0-20180221005942-b7d4802280ae
	github.com/lestrrat/go-pdebug v0.0.0-20180220043741-569c97477ae8 // indirect
	github.com/lyft/protoc-gen-validate v0.0.14
	github.com/miekg/dns v1.0.14
	github.com/mosn/binding v0.0.0-20200413092018-2b47bdb20a9f
	github.com/mosn/easygo v0.0.0-20201210062404-62796fdb3827
	github.com/mosn/registry v0.0.0-20210108061200-d7b63bc1904b
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/stretchr/testify v1.7.0
	github.com/tetratelabs/proxy-wasm-go-sdk v0.0.13
	github.com/trainyao/go-maglev v0.0.0-20200611125015-4c1ae64d96a8
	github.com/urfave/cli v1.20.0
	github.com/valyala/fasthttp v0.0.0-20200605121233-ac51d598dc54
	github.com/valyala/fasttemplate v1.1.0
	github.com/wasmerio/wasmer-go v1.0.3
	go.uber.org/atomic v1.7.0
	go.uber.org/automaxprocs v1.3.0
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/sys v0.0.0-20201024232916-9f70ab9862d5
	golang.org/x/tools v0.0.0-20191125144606-a911d9008d1f
	google.golang.org/genproto v0.0.0-20200305110556-506484158171
	google.golang.org/grpc v1.28.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	istio.io/api v0.0.0-20200227213531-891bf31f3c32
	k8s.io/klog v1.0.0
	mosn.io/api v0.0.0-20210204052134-5b9a826795fd
	mosn.io/pkg v0.0.0-20210209020457-6c79e1ef93f4
	mosn.io/proxy-wasm-go-host v0.0.0-20210310031752-6be0cc6caf8e
)

replace github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.9.4
