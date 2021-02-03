//module mosn.io/mosn
//module xxx
module gitlab.alipay-inc.com/golangextention

go 1.12

require (
	github.com/apache/dubbo-go v0.1.2-0.20200224151332-dd1a3c24d656 // indirect
	github.com/benbjohnson/clock v1.0.0 // indirect
	github.com/detailyang/easyreader-go v0.0.0-20200313073241-596c7a48c3d3 // indirect
	github.com/detailyang/keymutex-go v0.1.0 // indirect
	github.com/fsnotify/fsnotify v1.4.10-0.20200417215612-7f4cf4dd2b52
	github.com/gin-gonic/gin v1.5.0 // indirect
	github.com/labstack/echo/v4 v4.1.15 // indirect
	github.com/s12v/go-jwks v0.2.0 // indirect
	github.com/shirou/gopsutil v2.19.12+incompatible // indirect
	mosn.io/api v0.0.0-20210129030635-d7dc8206d7b7
	mosn.io/mosn v0.0.0-20210129123201-c2fcb7340e69
	mosn.io/pkg v0.0.0-20201228090327-daaf86502a50
)

replace (
	github.com/apache/dubbo-go-hessian2 => github.com/apache/dubbo-go-hessian2 v1.4.1-0.20200516085443-fa6429e4481d // perf: https://github.com/apache/dubbo-go-hessian2/pull/188
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.5
	google.golang.org/grpc => google.golang.org/grpc v1.28.0
)
