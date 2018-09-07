package gxzookeeper

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/AlexStocks/goext/database/registry"
	"github.com/AlexStocks/goext/log"
	"github.com/stretchr/testify/suite"
)

type WatcherTestSuite struct {
	suite.Suite
	sa   gxregistry.ServiceAttr
	node gxregistry.Node
	reg  gxregistry.Registry
	wt   gxregistry.Watcher
}

func (suite *WatcherTestSuite) SetupSuite() {
	suite.sa = gxregistry.ServiceAttr{
		Group:    "bjtelecom",
		Service:  "shopping",
		Protocol: "pb",
		Version:  "1.0.1",
		Role:     gxregistry.SRT_Consumer,
	}

	suite.node = gxregistry.Node{ID: "node0", Address: "127.0.0.1", Port: 12345}
}

func (suite *WatcherTestSuite) TearDownSuite() {
}

func (suite *WatcherTestSuite) SetupTest() {
	var err error

	suite.reg, err = NewRegistry(
		gxregistry.WithAddrs([]string{"127.0.0.1:2181"}...),
		gxregistry.WithTimeout(10e9),
		gxregistry.WithRoot("/test"),
	)
	suite.Equal(nil, err, "NewRegistry")
	suite.wt, err = suite.reg.Watch(
		gxregistry.WithWatchRoot("/test"),
		gxregistry.WithWatchFilter(gxregistry.ServiceAttr{
			Service: "shopping",
			Role:    gxregistry.SRT_Provider,
		}),
	)
	suite.Equal(nil, err, "NewRegistry")
}

func (suite *WatcherTestSuite) TearDownTest() {
	suite.wt.Close()
	suite.reg.Close()
}

func (suite *WatcherTestSuite) TestWatchService() {
	sa := gxregistry.ServiceAttr{
		Group:    "bjtelecom",
		Service:  "shopping",
		Protocol: "pb",
		Version:  "1.0.1",
		Role:     gxregistry.SRT_Provider,
	}

	// case1
	// 先注册两个 consumer 节点
	node := gxregistry.Node{ID: "node1", Address: "127.0.0.1", Port: 12341}
	consumerService := gxregistry.Service{
		Attr:  &suite.sa,
		Nodes: []*gxregistry.Node{&suite.node, &node},
	}
	err := suite.reg.Register(consumerService)
	gxlog.CInfo("register consumer service %s", consumerService)
	suite.Equalf(nil, err, "Register(service:%+v)", consumerService)

	// 注册两个 provider 节点
	node0 := gxregistry.Node{ID: "node0", Address: "127.0.0.1", Port: 22340}
	node1 := gxregistry.Node{ID: "node1", Address: "127.0.0.1", Port: 22341}
	providerService := gxregistry.Service{
		Attr:  &sa,
		Nodes: []*gxregistry.Node{&node0, &node1},
	}
	gxlog.CInfo("register provider service %s", providerService)
	err = suite.reg.Register(providerService)
	suite.Equalf(nil, err, "Register(service:%+v)", providerService)

	time.Sleep(3e9)

	// 注册一个 provider 节点
	node2 := gxregistry.Node{ID: "node2", Address: "127.0.0.1", Port: 22342}
	providerService = gxregistry.Service{
		Attr:  &sa,
		Nodes: []*gxregistry.Node{&node2},
	}
	gxlog.CInfo("register provider service %s", providerService)
	err = suite.reg.Register(providerService)
	suite.Equalf(nil, err, "Register(service:%+v)", providerService)

	time.Sleep(3e9)

	// 注册一个新的 provider service 路径
	sa.Service = "shopx"
	node3 := gxregistry.Node{ID: "node3", Address: "127.0.0.1", Port: 22343}
	providerService = gxregistry.Service{
		Attr:  &sa,
		Nodes: []*gxregistry.Node{&node3},
	}
	gxlog.CInfo("register provider service %s", providerService)
	err = suite.reg.Register(providerService)
	suite.Equalf(nil, err, "Register(service:%+v)", providerService)
	time.Sleep(5e9)
}

func (suite *WatcherTestSuite) TestValid() {
	time.Sleep(3e9)
	flag := suite.wt.Valid()
	suite.Equal(true, flag, "before watcher.Close()")
	suite.wt.Close()
	flag = suite.wt.Valid()
	suite.Equal(false, flag, "after watcher.Close()")
	flag = suite.wt.IsClosed()
	suite.Equal(true, flag, "after watcher.Close()")
}

// 在 sleep 语句中，重启 zookeeper 来测试注册数据可靠性
func (suite *WatcherTestSuite) TestZookeeperRestart() {
	var (
		err      error
		flag     bool
		node1    gxregistry.Node
		service  gxregistry.Service
		service1 []gxregistry.Service
	)

	node1 = gxregistry.Node{ID: "node1", Address: "127.0.0.1", Port: 12346}

	service = gxregistry.Service{
		Attr:  &suite.sa,
		Nodes: []*gxregistry.Node{&suite.node, &node1},
	}

	err = suite.reg.Register(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)

	flag = suite.wt.Valid()
	suite.Equal(true, flag, "before watcher.Close()")
	fmt.Println("Test zookeeper restart now. Please restart zookeeper in 7s.")

	//for i := 0; i < 20; i++ {
	//	time.Sleep(2e9)
	//	fmt.Printf("sleep %d seconds\n", (i+1)*2)
	//}

	service1, err = suite.reg.GetServices(suite.sa)
	suite.Equalf(nil, err, "registry.GetService(ServiceAttr:%+v)", suite.sa)
	suite.Equalf(1, len(service1[0].Nodes), "registry.GetService(ServiceAttr:%+v)", suite.sa)
	suite.Equalf(2, len(service1), "registry.GetService(ServiceAttr:%+v)", suite.sa)
}

func (suite *WatcherTestSuite) TestNotify() {
	var (
		err   error
		event *gxregistry.EventResult
		wg    sync.WaitGroup
		f     func()
		sa    gxregistry.ServiceAttr
		//node  gxregistry.Node
	)
	f = func() {
		defer wg.Done()
		event, err = suite.wt.Notify()
		suite.Equal(nil, err, "watch.Notify")
		fmt.Printf("event:%s, err:%+v\n", event.GoString(), err)
	}

	sa = gxregistry.ServiceAttr{
		Group:    "bjtelecom",
		Service:  "shopping",
		Protocol: "pb",
		Version:  "1.0.1",
		Role:     gxregistry.SRT_Provider,
	}

	service := gxregistry.Service{
		Attr:  &sa,
		Nodes: []*gxregistry.Node{&suite.node},
	}

	wg.Add(1)
	go f()
	err = suite.reg.Register(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)
	wg.Wait()
	suite.Equal(nil, err)
	suite.Equalf(gxregistry.ServiceAdd, event.Action, "event:%+v", event)
	suite.Equalf(service, *event.Service, "event:%+v", event)

	wg.Add(1)
	go f()
	service = gxregistry.Service{
		Attr:  &sa,
		Nodes: []*gxregistry.Node{&suite.node},
	}
	err = suite.reg.Deregister(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)
	wg.Wait()
	suite.Equal(nil, err)
	suite.Equalf(gxregistry.ServiceDel, event.Action, "event:%+v", event)
	suite.Equalf(service, *event.Service, "event:%+v", event)
}

func TestWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}
