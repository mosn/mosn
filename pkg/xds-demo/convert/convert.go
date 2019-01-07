// xds convert logic, split from package config/callback.go and config/convertxds.go
// can be extended in network filter and stream filter
package convert

import (
	"go/types"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	"github.com/alipay/sofa-mosn/pkg/log"
)

// 根据networkfilter的配置，获取到所有的stream filters配置(pb)
type GetStreamFiltersConfig func(s *types.Struct) map[string]*types.Struct

var getStreamFiltersConfigFuncs map[string]GetStreamFiltersConfig

func init() {
	getStreamFiltersConfigFuncs = make(map[string]GetStreamFiltersConfig)
	RegisterGetStreamFiltersConfig(xdsutil.HTTPConnectionManager, httpBaseGetStreamFiltersConfig)
	RegisterGetStreamFiltersConfig(v2.RPC_PROXY, httpBaseGetStreamFiltersConfig)
	RegisterGetStreamFiltersConfig(v2.X_PROXY, xProxyGetStreamFiltersConfig)
}

func RegisterGetStreamFiltersConfig(name string, f GetStreamFiltersConfig) {
	getStreamFiltersConfigFuncs[name] = f
}

func httpBaseGetStreamFiltersConfig(s *types.Struct) map[string]*types.Struct {
	filterConfig := &xdshttp.HttpConnectionManager{}

	xdsutil.StructToMessage(s, filterConfig)
	return filterConfig.GetHttpFilters()

}

func xProxyGetStreamFiltersConfig(s *types.Struct) map[string]*types.Struct {
	filterConfig := &xdsxproxy.XProxy{}
	xdsutil.StructToMessage(s, filterConfig)
	return filterConfig.GetStreamFilters()
}

func getStreamFiltersConfig(name string, s *types.Struct) map[string]*types.Struct {
	if f, ok := getStreamFiltersConfigFuncs[name]; ok {
		return f(s)
	}
	return nil
}

//  转换配置(pb)为MOSN的Stream filters配置
type HandleStreamFilterConfig func(s *types.Struct) v2.Filter

type handleStreamFilterConfigFuncs map[string]HandleStreamFilterConfig

func init() {
	handleStreamFilterConfigFuncs = make(map[string]HandleStreamFilterConfig)
	RegisterHandleStreamFilterConfig(v2.MIXER, convertMixerFilter)
	RegisterHandleStreamFilterConfig(v2.FaultStream, convertStreamFaultInjectConfig)
	RegisterHandleStreamFilterConfig(IstioFault, convertStreamFaultInjectConfig)
}

func RegisterHandleStreamFilterConfig(name string, f HandleStreamFilterConfig) {
	handleStreamFilterConfigFuncs[name] = f
}

// convertMixerFilter 和 convertStreamFaultInjectConfig 保持原来的逻辑
// 网关可以扩展自己的Stream Filter

func convertStreamFilters(configs map[string]*types.Struct) []v2.Filter {
	filters := []v2.Filter{}
	for name, cfg := range configs {
		if f, ok := streamFilterHandlers[name]; ok {
			streamFilter := f(cfg)
			if streamFilter.Type != "" {
				filters = append(filters, streamFilter)
			}
		}
	}
	return filters
}

// 需要做适配改动: Mixer解析PerRouteConfig需要按照StreamFilter的标准，目前的逻辑存在一些差异
func convertPerRouteConfig(xdsPerRouteConfig map[string]*types.Struct) map[string]interface{} {
	perRouteConfig := make(map[string]interface{}, 0)
	for key, config := range xdsPerRouteConfig {
		if f, ok := streamFilterHandlers[key]; ok {
			streamFilter := f(cfg)
			if streamFilter.Type != "" {
				perRouteConfig[key] = streamFilter.Config // Mixer这里有点细微差异
			}
		}
	}
}

func convertListenerConfig(xdsListener *xdsapi.Listener) *v2.Listener {
	if !isSupport(xdsListener) {
		return nil
	}

	listenerConfig := &v2.Listener{
		ListenerConfig: v2.ListenerConfig{
			Name:       xdsListener.GetName(),
			BindToPort: convertBindToPort(xdsListener.GetDeprecatedV1()),
			Inspector:  true,
			HandOffRestoredDestinationConnections: xdsListener.GetUseOriginalDst().GetValue(),
			AccessLogs:                            convertAccessLogs(xdsListener),
			LogPath:                               "stdout",
		},
		Addr: convertAddress(&xdsListener.Address),
		PerConnBufferLimitBytes: xdsListener.GetPerConnectionBufferLimitBytes().GetValue(),
		LogLevel:                uint8(log.INFO),
	}

	// virtual listener need none filters
	if listenerConfig.Name == "virtual" {
		return listenerConfig
	}

	listenerConfig.FilterChains = convertFilterChains(xdsListener.GetFilterChains())

	// 改动点
	if listenerConfig.FilterChains != nil &&
		len(listenerConfig.FilterChains) == 1 &&
		listenerConfig.FilterChains[0].Filters != nil {
		// listenerConfig.StreamFilters = convertStreamFilters(&xdsListener.FilterChains[0].Filters[0])
		networkFilter := xdsListener.FilterChains[0].Filters[0]
		streamFilters := getStreamFiltersConfig(networkFilter.GetName(), networkFilter.GetConfig())
		listenerConfig.StreamFilters = convertStreamFilters(streamFilters)

	}

	listenerConfig.DisableConnIo = GetListenerDisableIO(&listenerConfig.FilterChains[0])

	return listenerConfig
}

type HandleFilterAccessLog func(filter xdslistener.Filter) v2.AccessLog

func convertAccessLogs(xdsListener *xdsapi.Listener) []v2.AccessLog {
	if xdsListener == nil {
		return nil
	}

	accessLogs := make([]v2.AccessLog, 0)
	for _, xdsFilterChain := range xdsListener.GetFilterChains() {
		for _, xdsFilter := range xdsFilterChain.GetFilters() {
			name := xdsFilter.GetName()
			// handlFilterAccessLogFuns 和其他类似 就暂时先不写了
			if f, ok := handlFilterAccessLogFuns[name]; ok {
				accessLog := f(xdsFilter)
				accessLogs = append(accessLogs, accessLog)
			}
		}
	}
}

type HandleNetworkFilterConfig func(s *types.Struct) map[string]interface{}

func convertFilterConfig(name string, s *types.Struct) map[string]map[string]interface{} {
	if s == nil {
		return nil
	}

	filtersConfigParsed := make(map[string]map[string]interface{})
	// 类似,略

}

// 其他没有写的 保持原来的逻辑
