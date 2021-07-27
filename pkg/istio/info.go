package istio

import (
	_struct "github.com/golang/protobuf/ptypes/struct"
)

// IstioVersion adapt istio version
var IstioVersion = "unknow"

type XdsInfo struct {
	ServiceCluster string
	ServiceNode    string
	Metadata       *_struct.Struct
}

type TrafficInterceptionMode string

const (
	// InterceptionNone indicates that the workload is not using IPtables for traffic interception
	InterceptionNone TrafficInterceptionMode = "NONE"

	// InterceptionTproxy implies traffic intercepted by IPtables with TPROXY mode
	InterceptionTproxy TrafficInterceptionMode = "TPROXY"

	// InterceptionRedirect implies traffic intercepted by IPtables with REDIRECT mode
	// This is our default mode
	InterceptionRedirect TrafficInterceptionMode = "REDIRECT"
)

type Meta struct {
	// IstioVersion specifies the Istio version associated with the proxy
	IstioVersion string `json:"ISTIO_VERSION,omitempty"`

	// Labels specifies the set of workload instance (ex: k8s pod) labels associated with this node.
	Labels map[string]string `json:"LABELS,omitempty"`

	// InterceptionMode is the name of the metadata variable that carries info about
	// traffic interception mode at the proxy
	InterceptionMode TrafficInterceptionMode `json:"INTERCEPTION_MODE,omitempty"`

	// ClusterID defines the cluster the node belongs to.
	ClusterID string `json:"CLUSTER_ID,omitempty"`
}

var globalXdsInfo = &XdsInfo{}

// GetGlobalXdsInfo returns a struct, cannot be modified.
// Update XdsInfo by APIs.
func GetGlobalXdsInfo() XdsInfo {
	return *globalXdsInfo
}

func SetServiceCluster(sc string) {
	globalXdsInfo.ServiceCluster = sc
}

func SetServiceNode(sn string) {
	globalXdsInfo.ServiceNode = sn
}

func SetMetadata(meta *_struct.Struct) {
	globalXdsInfo.Metadata = meta
}
