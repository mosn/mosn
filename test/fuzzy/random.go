package fuzzy

import (
	"math/rand"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/mosn"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
)

type Server interface {
	// Close closes the server
	// If finished is true, the server cannot be restarted again.
	Close(finished bool)
	ReStart()
}

// Make sure server will be closed and restart
func FuzzyServer(stop chan struct{}, servers []Server, interval time.Duration) {
	min := interval / 10
	max := interval / 5
	for _, server := range servers {
		go func(server Server) {
			// random init state, state range [0,1]
			// state 0 means server will close next time
			// state 1 means server will restart next time
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			state := r.Intn(2)
			t := time.NewTicker(interval)
			defer t.Stop()
			for {
				select {
				case <-stop:
					return
				case <-t.C:
					time.Sleep(util.RandomDuration(min, max))
					if state == 0 {
						server.Close(false)
						state = 1
					} else {
						server.ReStart()
						state = 0
					}
				}
			}
		}(server)
	}
}

type Client interface {
	SendRequest()
}

func FuzzyClient(stop chan struct{}, client Client) {
	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				client.SendRequest()
				time.Sleep(util.RandomDuration(5*time.Millisecond, 100*time.Millisecond))
			}
		}
	}()
}

func CreateMeshProxy(t *testing.T, stop chan struct{}, serverList []string, proto types.Protocol) string {
	meshAddr := util.CurrentMeshAddr()
	cfg := util.CreateProxyMesh(meshAddr, serverList, proto)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-stop
		mesh.Close()
	}()
	time.Sleep(5 * time.Second)
	return meshAddr
}

func CreateMeshCluster(t *testing.T, stop chan struct{}, serverList []string, appproto, meshproto types.Protocol) string {
	meshAddr := util.CurrentMeshAddr()
	meshServerAddr := util.CurrentMeshAddr()
	cfg := util.CreateMeshToMeshConfig(meshAddr, meshServerAddr, appproto, meshproto, serverList, false)
	mesh := mosn.NewMosn(cfg)
	go mesh.Start()
	go func() {
		<-stop
		mesh.Close()
	}()
	time.Sleep(5 * time.Second)
	return meshAddr
}
