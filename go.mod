module mosn.io/mosn

go 1.12

require (
	bou.ke/monkey v1.0.2
	github.com/SkyAPM/go2sky v0.3.1-0.20200329092408-8b3e4d536d8d
	github.com/TarsCloud/TarsGo v0.0.0-20190521041003-0022231bc5b2
	github.com/alibaba/sentinel-golang v0.2.1-0.20200509115140-6d505e23ef30
	github.com/apache/dubbo-go-hessian2 v1.4.1-0.20200516085443-fa6429e4481d // perf: https://github.com/apache/dubbo-go-hessian2/pull/188
	github.com/c2h5oh/datasize v0.0.0-20171227191756-4eba002a5eae
	github.com/dchest/siphash v1.2.1
	github.com/envoyproxy/go-control-plane v0.9.4
	github.com/gin-gonic/gin v1.6.2 // indirect
	github.com/go-chi/chi v4.1.0+incompatible
	github.com/gogo/googleapis v1.3.2 // indirect
	github.com/gogo/protobuf v1.3.1
	github.com/golang/protobuf v1.3.5
	github.com/hashicorp/go-plugin v1.0.1
	github.com/json-iterator/go v1.1.9
	github.com/juju/errors v0.0.0-20190930114154-d42613fe1ab9
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/testing v0.0.0-20191001232224-ce9dec17d28b // indirect
	github.com/klauspost/cpuid v1.2.1 // indirect
	github.com/lyft/protoc-gen-validate v0.0.14
	github.com/miekg/dns v1.0.14
	github.com/mosn/binding v0.0.0-20200413092018-2b47bdb20a9f
	github.com/mosn/registry v0.0.0-20200612075445-e18906b5ec91
	github.com/neverhook/easygo v0.0.0-20180828090412-787757e64990
	github.com/openzipkin-contrib/zipkin-go-opentracing v0.3.5 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190812154241-14fe0d1b01d4
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563
	github.com/stretchr/testify v1.5.1
	github.com/tjfoc/gmsm v0.0.0-20190220013605-bfb01827afcb // indirect
	github.com/trainyao/go-maglev v0.0.0-20200611125015-4c1ae64d96a8
	github.com/urfave/cli v1.20.0
	github.com/valyala/fasthttp v0.0.0-20200605121233-ac51d598dc54
	github.com/valyala/fasttemplate v1.1.0
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d
	golang.org/x/net v0.0.0-20200324143707-d3edc9973b7e
	golang.org/x/sys v0.0.0-20200602100848-8d3cce7afc34
	google.golang.org/appengine v1.4.0 // indirect
	google.golang.org/grpc v1.28.0
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	istio.io/api v0.0.0-20200227213531-891bf31f3c32
	mosn.io/api v0.0.0-20200615113625-8eedc8a88c20
	mosn.io/pkg v0.0.0-20200428055827-06e02c6fbd6b
)

replace github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.9.4
