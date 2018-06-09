package cluster

import (
	"fmt"
	"sync"
	"testing"

	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func Test_cluster_refreshHealthHosts(t *testing.T) {
	
	 healthyHostPerLocality  := [][]types.Host{}
	
	healthyHostPerLocality[0][5] = nil
	type fields struct {
		initializationStarted          bool
		initializationCompleteCallback func()
		prioritySet                    *prioritySet
		info                           *clusterInfo
		mux                            sync.RWMutex
		initHelper                     concreteClusterInitHelper
		healthChecker                  types.HealthChecker
	}
	tests := []struct {
		name   string
		fields fields
	}{
	// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &cluster{
				initializationStarted:          tt.fields.initializationStarted,
				initializationCompleteCallback: tt.fields.initializationCompleteCallback,
				prioritySet:                    tt.fields.prioritySet,
				info:                           tt.fields.info,
				mux:                            tt.fields.mux,
				initHelper:                     tt.fields.initHelper,
				healthChecker:                  tt.fields.healthChecker,
			}
			c.refreshHealthHosts()
		})
	}
}
