package confreg

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
type PublishServiceResult struct {
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

//unpublish result
type UnPublishServiceResult struct {
	ErrorMessage string `json:"errorMessage"`
	Success      bool   `json:"success"`
}

//UnSubscribe request
type UnSubscribeServiceRequest struct {
	ServiceName string `json:"serviceName"`
}

//UnSubscribe result
type UnSubscribeServiceResult struct {
	ErrorMessage string `json:"errorMessage"`
	Success      bool   `json:"success"`
}
