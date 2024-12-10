module mosn.io/mosn

go 1.18

require (
	github.com/SkyAPM/go2sky v0.5.0
	github.com/TarsCloud/TarsGo v1.1.4
	github.com/alibaba/sentinel-golang v1.0.4
	github.com/alibaba/sentinel-golang/pkg/datasource/nacos v0.0.0-20241028104215-b8d94d173765
	github.com/apache/dubbo-go-hessian2 v1.10.2
	github.com/apache/thrift v0.13.0
	github.com/c2h5oh/datasize v0.0.0-20171227191756-4eba002a5eae
	github.com/cch123/supermonkey v1.0.1-0.20210420090843-d792ef7fb1d7
	github.com/cncf/udpa/go v0.0.0-20220112060539-c52dc94e7fbe
	github.com/cncf/xds/go v0.0.0-20230607035331-e9ce68804cb4
	github.com/dchest/siphash v1.2.1
	github.com/envoyproxy/go-control-plane v0.11.1-0.20230524094728-9239064ad72f
	github.com/ghodss/yaml v1.0.0
	github.com/go-chi/chi v4.1.0+incompatible
	github.com/go-resty/resty/v2 v2.6.0
	github.com/gogo/protobuf v1.3.2
	github.com/golang-jwt/jwt v3.2.1+incompatible
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.4
	github.com/google/cel-go v0.5.1
	github.com/hashicorp/go-plugin v1.0.1
	github.com/json-iterator/go v1.1.12
	github.com/juju/errors v1.0.0
	github.com/lestrrat/go-jwx v0.0.0-20180221005942-b7d4802280ae
	github.com/libp2p/go-reuseport v0.4.0
	github.com/miekg/dns v1.1.50
	github.com/nacos-group/nacos-sdk-go v1.0.8
	github.com/opentracing/opentracing-go v1.1.0
	github.com/opentrx/seata-golang/v2 v2.0.4
	github.com/openzipkin/zipkin-go v0.2.2
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.1
	github.com/prometheus/client_model v0.2.0
	github.com/rcrowley/go-metrics v0.0.0-20200313005456-10cdbea86bc0
	github.com/stretchr/testify v1.8.3
	github.com/tetratelabs/wabin v0.0.0-20230304001439-f6f874872834
	github.com/trainyao/go-maglev v0.0.0-20200611125015-4c1ae64d96a8
	github.com/uber/jaeger-client-go v2.25.0+incompatible
	github.com/urfave/cli v1.22.1
	github.com/valyala/fasthttp v1.40.0
	github.com/valyala/fasttemplate v1.1.0
	go.opentelemetry.io/otel v1.14.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.14.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.14.0
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.14.0
	go.opentelemetry.io/otel/sdk v1.14.0
	go.opentelemetry.io/otel/trace v1.14.0
	go.uber.org/atomic v1.7.0
	go.uber.org/automaxprocs v1.3.0
	golang.org/x/crypto v0.21.0
	golang.org/x/net v0.23.0
	golang.org/x/sys v0.18.0
	golang.org/x/tools v0.7.0
	google.golang.org/genproto v0.0.0-20230410155749-daa745c078e1
	google.golang.org/grpc v1.56.3
	google.golang.org/grpc/examples v0.0.0-20210818220435-8ab16ef276a3
	google.golang.org/protobuf v1.33.0
	istio.io/api v0.0.0-20211103171850-665ed2b92d52
	istio.io/gogo-genproto v0.0.0-20210113155706-4daf5697332f
	k8s.io/klog v1.0.0
	mosn.io/api v1.6.0
	mosn.io/envoy-go-extension v0.0.0-20221230083446-5b8dd13435c7
	mosn.io/holmes v1.1.0
	mosn.io/pkg v1.6.0
	mosn.io/proxy-wasm-go-host v0.2.1-0.20230626122511-25a9e133320e
	vimagination.zapto.org/byteio v0.0.0-20200222190125-d27cba0f0b10
)

require (
	github.com/HdrHistogram/hdrhistogram-go v1.1.2 // indirect
	github.com/Shopify/sarama v1.19.0 // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.18 // indirect
	github.com/andybalholm/brotli v1.0.4 // indirect
	github.com/antlr/antlr4 v0.0.0-20200503195918-621b933c7a7f // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/buger/jsonparser v0.0.0-20181115193947-bf1c66bbce23 // indirect
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dubbogo/getty v1.3.4 // indirect
	github.com/dubbogo/go-zookeeper v1.0.3 // indirect
	github.com/dubbogo/gost v1.11.16 // indirect
	github.com/eapache/go-resiliency v1.1.0 // indirect
	github.com/eapache/go-xerial-snappy v0.0.0-20180814174437-776d5712da21 // indirect
	github.com/eapache/queue v1.1.0 // indirect
	github.com/envoyproxy/protoc-gen-validate v0.10.1 // indirect
	github.com/go-errors/errors v1.0.1 // indirect
	github.com/go-logr/logr v1.2.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.5 // indirect
	github.com/go-playground/locales v0.13.0 // indirect
	github.com/go-playground/universal-translator v0.17.0 // indirect
	github.com/go-playground/validator/v10 v10.4.1 // indirect
	github.com/golang/snappy v0.0.3 // indirect
	github.com/google/uuid v1.3.0 // indirect
	github.com/gorilla/websocket v1.4.2 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.11.3 // indirect
	github.com/hashicorp/go-hclog v0.0.0-20180709165350-ff2cf002a8dd // indirect
	github.com/hashicorp/go-syslog v1.0.0 // indirect
	github.com/hashicorp/yamux v0.0.0-20180604194846-3520598351bb // indirect
	github.com/jinzhu/copier v0.3.2 // indirect
	github.com/jmespath/go-jmespath v0.0.0-20180206201540-c2b33e8439af // indirect
	github.com/k0kubun/pp v3.0.1+incompatible // indirect
	github.com/klauspost/compress v1.15.11 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/leodido/go-urn v1.2.0 // indirect
	github.com/lestrrat/go-file-rotatelogs v0.0.0-20180223000712-d3151e2a480f // indirect
	github.com/lestrrat/go-pdebug v0.0.0-20180220043741-569c97477ae8 // indirect
	github.com/lestrrat/go-strftime v0.0.0-20180220042222-ba3bf9c1d042 // indirect
	github.com/magiconair/properties v1.8.1 // indirect
	github.com/mattn/go-colorable v0.1.7 // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.1 // indirect
	github.com/mitchellh/go-testing-interface v1.0.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/common v0.26.0 // indirect
	github.com/prometheus/procfs v0.6.0 // indirect
	github.com/russross/blackfriday/v2 v2.0.1 // indirect
	github.com/satori/go.uuid v1.2.0 // indirect
	github.com/shirou/gopsutil v3.20.11+incompatible // indirect
	github.com/shirou/gopsutil/v3 v3.21.6 // indirect
	github.com/shurcooL/sanitized_anchor_name v1.0.0 // indirect
	github.com/stretchr/objx v0.5.0 // indirect
	github.com/tetratelabs/wazero v1.2.1 // indirect
	github.com/tklauser/go-sysconf v0.3.6 // indirect
	github.com/tklauser/numcpus v0.2.2 // indirect
	github.com/toolkits/concurrent v0.0.0-20150624120057-a4371d70e3e3 // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/ugorji/go/codec v1.1.7 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/wasmerio/wasmer-go v1.0.4 // indirect
	go.opentelemetry.io/otel/exporters/otlp/internal/retry v1.14.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.14.0 // indirect
	go.opentelemetry.io/proto/otlp v0.19.0 // indirect
	go.uber.org/multierr v1.6.0 // indirect
	go.uber.org/zap v1.17.0 // indirect
	golang.org/x/arch v0.0.0-20200826200359-b19915210f00 // indirect
	golang.org/x/mod v0.9.0 // indirect
	golang.org/x/term v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	gopkg.in/ini.v1 v1.51.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.0.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	vimagination.zapto.org/memio v1.0.0 // indirect
)

replace github.com/envoyproxy/go-control-plane => github.com/envoyproxy/go-control-plane v0.10.0

replace istio.io/api => istio.io/api v0.0.0-20211103171850-665ed2b92d52
