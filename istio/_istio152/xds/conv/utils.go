package conv

import (
	"time"

	xdslistener "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	xdsaccesslog "github.com/envoyproxy/go-control-plane/envoy/config/accesslog/v2"
	accesslog "github.com/envoyproxy/go-control-plane/envoy/config/filter/accesslog/v2"
	xdshttp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	xdstcp "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/tcp_proxy/v2"
	"github.com/envoyproxy/go-control-plane/pkg/conversion"
	"github.com/golang/protobuf/proto"
	"github.com/golang/protobuf/ptypes"
	"github.com/golang/protobuf/ptypes/duration"
	"mosn.io/mosn/pkg/log"
)

func getFilterConfig(filter *xdslistener.Filter, out proto.Message) error {
	switch c := filter.ConfigType.(type) {
	case *xdslistener.Filter_Config:
		if err := conversion.StructToMessage(c.Config, out); err != nil {
			return err
		}
	case *xdslistener.Filter_TypedConfig:
		if err := ptypes.UnmarshalAny(c.TypedConfig, out); err != nil {
			return err
		}
	}
	return nil
}

func GetHTTPConnectionManager(filter *xdslistener.Filter) *xdshttp.HttpConnectionManager {
	cm := &xdshttp.HttpConnectionManager{}
	if err := getFilterConfig(filter, cm); err != nil {
		log.DefaultLogger.Errorf("failed to get HTTP connection manager config: %s", err)
		return nil
	}
	return cm
}

func GetTcpProxy(filter *xdslistener.Filter) *xdstcp.TcpProxy {
	cm := &xdstcp.TcpProxy{}
	if err := getFilterConfig(filter, cm); err != nil {
		log.DefaultLogger.Errorf("failed to get HTTP connection manager config: %s", err)
		return nil
	}
	return cm
}

func GetAccessLog(log *accesslog.AccessLog) (*xdsaccesslog.FileAccessLog, error) {
	al := &xdsaccesslog.FileAccessLog{}
	switch log.ConfigType.(type) {
	case *accesslog.AccessLog_Config:
		if err := conversion.StructToMessage(log.GetConfig(), al); err != nil {
			return nil, err
		}

	case *accesslog.AccessLog_TypedConfig:
		if err := ptypes.UnmarshalAny(log.GetTypedConfig(), al); err != nil {
			return nil, err
		}
	}
	return al, nil

}

func ConvertDuration(p *duration.Duration) time.Duration {
	if p == nil {
		return time.Duration(0)
	}
	d := time.Duration(p.Seconds) * time.Second
	if p.Nanos != 0 {
		dur := d + time.Duration(p.Nanos)
		if (dur < 0) != (p.Nanos < 0) {
			log.DefaultLogger.Warnf("duration: %#v is out of range for time.Duration, ignore nanos", p)
		} else {
			d = dur
		}
	}
	return d
}
