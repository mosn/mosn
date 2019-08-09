package tenant

import "sofastack.io/sofa-mosn/pkg/istio/utils"

type attributesBuilderPlugin struct{}

func (m attributesBuilderPlugin) AddAttributes(builder *utils.AttributesBuilder) {
	tenantInfo := GetTenantInfo()
	for k, v := range tenantInfo {
		builder.AddString(k, v)
	}
}

func NewTenantAttributesBuilderPlugin() attributesBuilderPlugin {
	return attributesBuilderPlugin{}
}
