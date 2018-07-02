package cluster

import (
	"testing"

	"gitlab.alipay-inc.com/afe/mosn/pkg/api/v2"
	"gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

func Test_roundRobinLoadBalancer_ChooseHost(t *testing.T) {

	host1 := NewHost(v2.Host{Address: "127.0.0.1", Hostname: "test", Weight: 0}, nil)
	host2 := NewHost(v2.Host{Address: "127.0.0.2", Hostname: "test2", Weight: 0}, nil)
	host3 := NewHost(v2.Host{Address: "127.0.0.3", Hostname: "test", Weight: 0}, nil)
	host4 := NewHost(v2.Host{Address: "127.0.0.4", Hostname: "test2", Weight: 0}, nil)
	host5 := NewHost(v2.Host{Address: "127.0.0.5", Hostname: "test2", Weight: 0}, nil)

	hosts1 := []types.Host{host1, host2}
	hosts2 := []types.Host{host3, host4}
	hosts3:= []types.Host{host5}
	

	hs1 := hostSet{
		hosts: hosts1,
	}

	hs2 := hostSet{
		hosts: hosts2,
	}
	
	hs3 := hostSet{
		hosts:hosts3,
	}

	hostset := []types.HostSet{&hs1, &hs2,&hs3}

	prioritySet := prioritySet{
		hostSets: hostset,
	}

	loadbalaner := loadbalaner{
		prioritySet: &prioritySet,
	}

	l := &roundRobinLoadBalancer{
		loadbalaner: loadbalaner,
	}

	want := []types.Host{host1, host2, host3, host4,host5}

	for i := 0; i < len(want); i++ {
		got := l.ChooseHost(nil)
		if got != want[i] {
			t.Errorf("Test Error in case %d , got %+v, but want %+v,", i, got, want[i])
		}
	}
}