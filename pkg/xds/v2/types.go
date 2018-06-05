package v2

import "net/http"

type xdsClient struct {
	httpClient *http.Client
	serviceCluster string
	serviceNode string
	//logger log.logger
}