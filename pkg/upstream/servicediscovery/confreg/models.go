package registry

type ApplicationInfoRequest struct {
	AntShareCloud    bool   `json:"antShareCloud"`
	DataCenter       string `json:"dataCenter"`
	AppName          string `json:"appName"`
	Zone             string `json:"zone"`
	RegistryEndpoint string ` json:"registryEndpoint"`
	AccessKey        string `json:"accessKey"`
	SecretKey        string `json:"secretKey"`
}

//pub request
type PublishServiceRequest struct {
	ServiceName      string           `json:"serviceName"`
	ProviderMetaInfo ProviderMetaInfo `json:"providerMetaInfo"`
}

type ProviderMetaInfo struct {
	Protocol      string `json:"protocol"`
	Version       string `json:"version"`
	SerializeType string `json:"serializeType"`
	AppName       string `json:"appName"`
}

//pub result
type HttpResponse struct {
	ErrorMessage string `json:"errorMessage"`
	Success      bool   `json:"success"`
}

// sub request
type SubscribeServiceRequest struct {
	ServiceName string `json:"serviceName"`
}

//sub result
type SubscribeServiceResult struct {
	ErrorMessage string   `json:"errorMessage"`
	Success      bool     `json:"success"`
	ServiceName  string   `json:"serviceName"`
	Datas        []string `json:"datas"`
}

//unpublish request
type UnPublishServiceRequest struct {
	ServiceName string `json:"serviceName"`
}

//UnSubscribe request
type UnSubscribeServiceRequest struct {
	ServiceName string `json:"serviceName"`
}
