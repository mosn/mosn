package v2

type LeastRequestLbConfig struct {
	ChoiceCount uint32
}

func (lbconfig *LeastRequestLbConfig) isCluster_LbConfig() {
}

type IsCluster_LbConfig interface {
	isCluster_LbConfig()
}
