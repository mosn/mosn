package gxetcd

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/AlexStocks/goext/database/registry"
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
		// gxregistry.WithAddrs([]string{"127.0.0.1:2379", "127.0.0.1:12379", "127.0.0.1:22379"}...),
		gxregistry.WithAddrs([]string{"127.0.0.1:2379"}...),
		gxregistry.WithTimeout(10e9),
		gxregistry.WithRoot("/etcd_test"),
	)
	suite.Equal(nil, err, "NewRegistry")
	suite.wt, err = suite.reg.Watch(
		gxregistry.WithWatchRoot("/etcd_test"),
	)
	suite.Equal(nil, err, "NewRegistry")
}

func (suite *WatcherTestSuite) TearDownTest() {
	suite.wt.Close()
	suite.reg.Close()
}

func (suite *WatcherTestSuite) TestValid() {
	flag := suite.wt.Valid()
	suite.Equal(true, flag, "before watcher.Close()")
	suite.wt.Close()
	flag = suite.wt.Valid()
	suite.Equal(false, flag, "after watcher.Close()")
	flag = suite.wt.IsClosed()
	suite.Equal(true, flag, "after watcher.Close()")
}

// 本单测测试步骤：
// step1: 注释掉Registry的handleEtcdRestart;
// step2: 测试开始后stop etcd 6s以上再启动etcd；
//
// 在2s以内立即重启etcd，输出结果如下：
// ttl:9
// ttl:7
// ttl:8
// ttl:6
// ttl:10
// ttl:7
// ttl:9
// ttl:7
// ttl:9
// ttl:7
//
// 在5s以上重启etcd，输出结果如下：
// ttl:9
// ttl:7
// ttl:9
// ttl:10
// ttl:8
// ttl:6
// ttl:4
// ttl:2
// ttl:0
// ttl:-1
// ttl:-1
// ttl:-1
//
// 从以上输出可见：etcdv3 底层的 client 是有重连机制的，但是其相关 lease 可能失效，
// 那么与 lease 相关的注册信息也会随之失效
func (suite *WatcherTestSuite) TestEtcdRestart() {
	var (
		err      error
		flag     bool
		service  gxregistry.Service
		service1 []gxregistry.Service
	)

	service = gxregistry.Service{
		Attr:  &suite.sa,
		Nodes: []*gxregistry.Node{&suite.node},
	}

	err = suite.reg.Register(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)

	flag = suite.wt.Valid()
	suite.Equal(true, flag, "before watcher.Close()")
	fmt.Println("Test etcd restart now. Please restart etcd in 7s.")

	for i := 0; i < 7; i++ {
		fmt.Printf("time interval:%d, ttl:%d\n", i*2, suite.wt.(*Watcher).client.TTL())
		time.Sleep(2e9)
	}

	//suite.execBashCommand("cd bin/etcd-cluster/single/ && sh load.sh stop")
	fmt.Println("bash command: cd bin/etcd-cluster/single/ && sh load.sh stop")
	time.Sleep(8e9)
	//suite.execBashCommand("cd bin/etcd-cluster/single/ && sh load.sh start")
	fmt.Println("bash command: cd bin/etcd-cluster/single/ && sh load.sh start")

	for i := 0; i < 4; i++ {
		fmt.Printf("time interval:%d, ttl:%d\n", i*2, suite.wt.(*Watcher).client.TTL())
		time.Sleep(2e9)
	}

	service1, err = suite.reg.GetServices(suite.sa)
	suite.Equalf(nil, err, "registry.GetService(ServiceAttr:%+v)", suite.sa)
	suite.Equalf(1, len(service1[0].Nodes), "registry.GetService(ServiceAttr:%+v)", suite.sa)
	//suite.Equalf(service, *service1, "registry.GetService(ServiceAttr:%+v) = ", suite.sa)
}

func (suite *WatcherTestSuite) TestNotify() {
	var (
		err   error
		event *gxregistry.EventResult
		wg    sync.WaitGroup
		f     func()
		//node  gxregistry.Node
	)
	f = func() {
		defer wg.Done()
		event, err = suite.wt.Notify()
		suite.Equal(nil, err, "watch.Notify")
		//fmt.Printf("ttl:%d, event:%s, err:%+v\n", suite.wt.(*Watcher).client.TTL(), event.GoString(), err)
	}

	service := gxregistry.Service{
		Attr:  &suite.sa,
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
		Attr:  &suite.sa,
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
