package tenant

import (
	"strings"
	"sofastack.io/sofa-mosn/pkg/log"
	"sofastack.io/sofa-mosn/pkg/istio/control/http"
	"sofastack.io/sofa-mosn/pkg/istio/mixerclient"
)

const (
	tenancySeparator     = "||"
	tenancyKVSeparator   = "="
	serviceNodeSeparator = "~"
)

var tenantInfoMap map[string]string

func Init(serviceNode string) {
	initTenantInfo(serviceNode)
	initTenantAttributesBuilderPlugin()
}

func initTenantInfo(serviceNode string) {
	nodeWithTenancy := strings.Split(serviceNode, tenancySeparator)
	if len(nodeWithTenancy) <= 1 {
		log.DefaultLogger.Warnf("[tenant] [Init] AntCloudTenancy Info is empty, serviceNode=%s", serviceNode)
		return
	}

	tenancy := nodeWithTenancy[1]
	tParts := strings.Split(tenancy, serviceNodeSeparator)
	infoMap := make(map[string]string, len(tParts))
	for _, part := range tParts {
		kv := strings.Split(part, tenancyKVSeparator)
		if len(kv) == 2 {
			infoMap[kv[0]] = kv[1]
			mixerclient.AddAttributeToGlobalList(kv[0])
		}
	}
	tenantInfoMap = infoMap

	log.DefaultLogger.Infof("[tenant] [Init] ServiceNodeInfo init completed serviceNode=%s, labels=%+v", serviceNode, infoMap)
}

func initTenantAttributesBuilderPlugin() {
	http.RegisterAttributesBuilderPlugin(NewTenantAttributesBuilderPlugin())
}

func GetTenantInfo() map[string]string {
	result := make(map[string]string, len(tenantInfoMap))
	for k, v := range tenantInfoMap {
		result[k] = v
	}
	return result
}
