package v2

import (
	"mosn.io/api"
)

type LeastRequestLbConfig struct {
	ChoiceCount uint32
}

func (lbconfig *LeastRequestLbConfig) isCluster_LbConfig() {
}

type IsCluster_LbConfig interface {
	isCluster_LbConfig()
}

type HashPolicy struct {
	Header     *HeaderHashPolicy     `json:"header,omitempty"`
	HttpCookie *HttpCookieHashPolicy `json:"http_cookie,omitempty"`
	SourceIP   *SourceIPHashPolicy   `json:"source_ip,omitempty"`
}

type HeaderHashPolicy struct {
	Key string `json:"key,omitempty"`
}

type HttpCookieHashPolicy struct {
	Name string             `json:"name,omitempty"`
	Path string             `json:"path,omitempty"`
	TTL  api.DurationConfig `json:"ttl,omitempty"`
}

type SourceIPHashPolicy struct {
}
