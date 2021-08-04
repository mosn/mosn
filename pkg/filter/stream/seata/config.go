package seata

import (
	"google.golang.org/grpc/keepalive"
	"time"
)

// TransactionInfo transaction info config
type TransactionInfo struct {
	RequestPath string `yaml:"requestPath" json:"requestPath"`
	Timeout     int32  `yaml:"timeout" json:"timeout"`
}

// TCCResource tcc resource config
type TCCResource struct {
	PrepareRequestPath  string `yaml:"prepareRequestPath" json:"prepareRequestPath"`
	CommitRequestPath   string `yaml:"commitRequestPath" json:"commitRequestPath"`
	RollbackRequestPath string `yaml:"rollbackRequestPath" json:"rollbackRequestPath"`
}

// Seata seata config
type Seata struct {
	Addressing         string `yaml:"addressing" json:"addressing"`
	ServerAddressing   string `yaml:"serverAddressing" json:"serverAddressing"`
	CommitRetryCount   int32  `default:"5" yaml:"commitRetryCount" json:"commitRetryCount,omitempty"`
	RollbackRetryCount int32  `default:"5" yaml:"rollbackRetryCount" json:"rollbackRetryCount,omitempty"`

	ClientParameters struct {
		Time                time.Duration `yaml:"time" json:"time"`
		Timeout             time.Duration `yaml:"timeout" json:"timeout"`
		PermitWithoutStream bool          `yaml:"permitWithoutStream" json:"permitWithoutStream"`
	} `yaml:"clientParameters" json:"clientParameters"`

	TransactionInfos []*TransactionInfo `yaml:"transactionInfos" json:"transactionInfos"`
	TCCResources     []*TCCResource     `yaml:"tccResources" json:"tccResources"`
}

// GetClientParameters used to config grpc connection keep alive
func (config *Seata) GetClientParameters() keepalive.ClientParameters {
	cp := keepalive.ClientParameters{
		Time:                10 * time.Second,
		Timeout:             time.Second,
		PermitWithoutStream: true,
	}
	if config.ClientParameters.Time > 0 {
		cp.Time = config.ClientParameters.Timeout
	}
	if config.ClientParameters.Timeout > 0 {
		cp.Timeout = config.ClientParameters.Timeout
	}
	cp.PermitWithoutStream = config.ClientParameters.PermitWithoutStream
	return cp
}

