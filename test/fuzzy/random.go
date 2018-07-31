package fuzzy

import (
	"math/rand"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/cmd/mosn"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	"github.com/alipay/sofa-mosn/test/util"
)

type Server interface {
	Close()
	ReStart()
	GetID() string
}

func FuzzyServer(stop chan struct{}, servers []Server, interval time.Duration) {
	for _, server := range servers {
		go func(server Server) {
			t := time.NewTicker(interval)
			defer t.Stop()
			for {
				select {
				case <-stop:
					server.Close()
					return
				case <-t.C:
					time.Sleep(util.RandomDuration(100*time.Millisecond, time.Second))
					switch rand.Intn(3) {
					case 0:
						log.StartLogger.Infof("[FUZZY TEST] server close #%s", server.GetID())
						server.Close()
					default:
						server.ReStart()
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
