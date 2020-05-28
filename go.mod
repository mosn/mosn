module mosn.io/mosn

go 1.12

require (
	github.com/SkyAPM/go2sky v0.3.1-0.20200329092408-8b3e4d536d8d
	github.com/TarsCloud/TarsGo v0.0.0-20181112071624-2d42457f2025
	github.com/alibaba/sentinel-golang v0.2.1-0.20200509115140-6d505e23ef30
	github.com/apache/dubbo-go-hessian2 v1.5.0
	github.com/c2h5oh/datasize v0.0.0-20171227191756-4eba002a5eae
	github.com/envoyproxy/go-control-plane v0.8.0
	github.com/gogo/googleapis v1.2.0 // indirect
	github.com/gogo/protobuf v1.2.1
	github.com/golang/protobuf v1.3.2
	github.com/hashicorp/go-plugin v1.0.1
	github.com/json-iterator/go v1.1.7
	github.com/juju/errors v0.0.0-20190207033735-e65537c515d7
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8 // indirect
	github.com/juju/testing v0.0.0-20191001232224-ce9dec17d28b // indirect
	github.com/klauspost/compress v1.7.5 // indirect
	github.com/klauspost/cpuid v1.2.1 // indirect
	github.com/lyft/protoc-gen-validate v0.0.14
	github.com/miekg/dns v1.1.29
	github.com/neverhook/easygo v0.0.0-20180828090412-787757e64990
	github.com/pkg/errors v0.8.1
	github.com/prometheus/client_golang v1.1.0
	github.com/prometheus/client_model v0.0.0-20190129233127-fd36f4220a90
	github.com/rcrowley/go-metrics v0.0.0-20190826022208-cac0b30c2563
	github.com/stretchr/testify v1.5.1
	github.com/urfave/cli v1.20.0
	github.com/valyala/fasthttp v1.2.0
	golang.org/x/crypto v0.0.0-20200221231518-2aa609cf4a9d
	golang.org/x/net v0.0.0-20200226121028-0de0cce0169b
	golang.org/x/sys v0.0.0-20200223170610-d5e6a3e2c0ae
	google.golang.org/genproto v0.0.0-20190801165951-fa694d86fc64 // indirect
	google.golang.org/grpc v1.22.1
	gopkg.in/mgo.v2 v2.0.0-20190816093944-a6b53ec6cb22 // indirect
	istio.io/api v0.0.0-20190408162927-e9ab8d6a54a6
	mosn.io/api v0.0.0-20200416082846-2e7ce9a85557
	mosn.io/pkg v0.0.0-20200428055827-06e02c6fbd6b
)

replace github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.6.9
