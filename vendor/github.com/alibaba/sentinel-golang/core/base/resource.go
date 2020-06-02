package base

import "fmt"

// ResourceType represents classification of the resources
type ResourceType int32

const (
	ResTypeCommon ResourceType = iota
	ResTypeWeb
	ResTypeRPC
	ResTypeAPIGateway
	ResTypeDBSQL
	ResTypeCache
	ResTypeMQ
)

// TrafficType describes the traffic type: Inbound or Outbound
type TrafficType int32

const (
	// Inbound represents the inbound traffic (e.g. provider)
	Inbound TrafficType = iota
	// Outbound represents the outbound traffic (e.g. consumer)
	Outbound
)

func (t TrafficType) String() string {
	switch t {
	case Inbound:
		return "Inbound"
	case Outbound:
		return "Outbound"
	default:
		return fmt.Sprintf("%d", t)
	}
}

// ResourceWrapper represents the invocation
type ResourceWrapper struct {
	// global unique resource name
	name string
	// resource classification
	classification ResourceType
	// Inbound or Outbound
	flowType TrafficType
}

func (r *ResourceWrapper) String() string {
	return fmt.Sprintf("ResourceWrapper{name=%s, flowType=%s, classification=%d}", r.name, r.flowType, r.classification)
}

func (r *ResourceWrapper) Name() string {
	return r.name
}

func (r *ResourceWrapper) Classification() ResourceType {
	return r.classification
}

func (r *ResourceWrapper) FlowType() TrafficType {
	return r.flowType
}

func NewResourceWrapper(name string, classification ResourceType, flowType TrafficType) *ResourceWrapper {
	return &ResourceWrapper{name: name, classification: classification, flowType: flowType}
}
