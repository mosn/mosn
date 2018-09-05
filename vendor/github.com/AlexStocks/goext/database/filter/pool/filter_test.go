package gxpool

import (
	"testing"
	"time"
)

import (
	"github.com/AlexStocks/goext/database/filter"
	"github.com/AlexStocks/goext/database/registry"
	//"github.com/AlexStocks/goext/database/registry/etcdv3"
	"github.com/AlexStocks/goext/database/registry/zookeeper"
	"github.com/AlexStocks/goext/log"
	"github.com/stretchr/testify/suite"
)

type FilterTestSuite struct {
	suite.Suite
	root      string
	etcdAddrs []string
	pSA       gxregistry.ServiceAttr
	cSA       gxregistry.ServiceAttr
	nodes     []gxregistry.Node
	ttl       time.Duration
	reg       gxregistry.Registry
	filter    gxfilter.Filter
}

func (suite *FilterTestSuite) SetupSuite() {
	suite.root = "/test"

	// suite.etcdAddrs = []string{"127.0.0.1:2379"} // etcd
	suite.etcdAddrs = []string{"127.0.0.1:2181"} // zookeeper

	suite.pSA = gxregistry.ServiceAttr{
		Group:    "bjtelecom",
		Service:  "shopping",
		Protocol: "pb",
		Version:  "1.0.1",
		Role:     gxregistry.SRT_Provider,
	}

	suite.cSA = gxregistry.ServiceAttr{
		Group:    "bjtelecom",
		Service:  "shopping",
		Protocol: "pb",
		Version:  "1.0.1",
		Role:     gxregistry.SRT_Consumer,
	}

	suite.nodes = []gxregistry.Node{
		{ID: "node0", Address: "127.0.0.1", Port: 12345},
		{ID: "node1", Address: "127.0.0.2", Port: 12345},
		{ID: "node2", Address: "127.0.0.3", Port: 12345},
	}

	suite.ttl = 15e9
}

func (suite *FilterTestSuite) TearDownSuite() {
}

func (suite *FilterTestSuite) SetupTest() {
	var err error
	//suite.reg, err = gxetcd.NewRegistry(
	suite.reg, err = gxzookeeper.NewRegistry(
		gxregistry.WithAddrs(suite.etcdAddrs...),
		gxregistry.WithTimeout(3e9),
		gxregistry.WithRoot(suite.root),
	)
	suite.Equal(nil, err, "NewRegistry()")

	suite.filter, err = NewFilter(
		gxfilter.WithRegistry(suite.reg),
		WithTTL(10e9),
	)
	suite.Equal(nil, err, "NewFilter()")
}

func (suite *FilterTestSuite) TearDownTest() {
	suite.filter.Close()
	suite.reg.Close()
}

func (suite *FilterTestSuite) TestFilter_Options() {
	opts := suite.filter.Options()
	suite.Equal(suite.reg, opts.Registry)
}

func (suite *FilterTestSuite) TestFilter_get() {
	filter := suite.filter.(*Filter)

	// register suite.node
	service := gxregistry.Service{Attr: &suite.pSA, Nodes: []*gxregistry.Node{&suite.nodes[0], &suite.nodes[1]}}
	err := suite.reg.Register(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)
	service = gxregistry.Service{Attr: &suite.cSA, Nodes: []*gxregistry.Node{&suite.nodes[2]}}
	err = suite.reg.Register(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)
	time.Sleep(3e9)

	attr := gxregistry.ServiceAttr{Service: "shopping", Role: gxregistry.SRT_Provider}
	svcArr, err := filter.get(attr)
	suite.Equal(nil, err)
	suite.NotEqual(0, svcArr.Active)
	suite.Equal(2, len(svcArr.Arr))

	flag := suite.filter.CheckServiceAlive(attr, svcArr)
	suite.Equal(true, flag)

	// delete consumer service
	service = gxregistry.Service{Attr: &suite.pSA, Nodes: []*gxregistry.Node{&suite.nodes[0]}}
	err = suite.reg.Deregister(service)
	suite.Equalf(nil, err, "Deregister(service:%+v)", service)
	time.Sleep(3e9)

	// check token alive
	flag = suite.filter.CheckServiceAlive(attr, svcArr)
	suite.Equal(false, flag)

	// get alive again
	attr = gxregistry.ServiceAttr{Service: "shopping", Role: gxregistry.SRT_Provider}
	svcArr, err = filter.get(attr)
	suite.Equal(nil, err)
	suite.NotEqual(0, svcArr.Active)
	suite.Equal(1, len(svcArr.Arr))

	service = gxregistry.Service{Attr: &suite.pSA, Nodes: []*gxregistry.Node{&suite.nodes[1]}}
	//service1, _ := balancer(uint64(0))
	suite.Equal(*(svcArr.Arr[0]), service)
}
func (suite *FilterTestSuite) TestRegistry_EtcdRestart() {
	filter := suite.filter.(*Filter)

	// register suite.node
	service := gxregistry.Service{Attr: &suite.pSA, Nodes: []*gxregistry.Node{&suite.nodes[0], &suite.nodes[1]}}
	err := suite.reg.Register(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)
	service = gxregistry.Service{Attr: &suite.cSA, Nodes: []*gxregistry.Node{&suite.nodes[2]}}
	err = suite.reg.Register(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)
	time.Sleep(3e9)

	attr := gxregistry.ServiceAttr{Service: "shopping", Role: gxregistry.SRT_Provider}
	serviceArray, err := filter.get(attr)
	suite.Equal(nil, err)
	suite.NotEqual(0, serviceArray.Active.UnixNano())
	suite.Equal(2, len(serviceArray.Arr))

	flag := suite.filter.CheckServiceAlive(attr, serviceArray)
	suite.Equal(true, flag)

	gxlog.CError("\n\nPls stop the etcd now...\n\n")
	time.Sleep(20e9)
	gxlog.CError("\n\nPls start the etcd now...\n\n")
	time.Sleep(20e9)

	// delete consumer service
	service = gxregistry.Service{Attr: &suite.pSA, Nodes: []*gxregistry.Node{&suite.nodes[0]}}
	err = suite.reg.Deregister(service)
	suite.Equalf(nil, err, "Deregister(service:%+v)", service)
	time.Sleep(3e9)

	// check token alive
	flag = suite.filter.CheckServiceAlive(attr, serviceArray)
	suite.Equal(false, flag)

	// get alive again
	attr = gxregistry.ServiceAttr{Service: "shopping", Role: gxregistry.SRT_Provider}
	serviceArray, err = filter.get(attr)
	suite.Equal(nil, err)
	suite.Equal(1, len(serviceArray.Arr))

	service = gxregistry.Service{Attr: &suite.pSA, Nodes: []*gxregistry.Node{&suite.nodes[1]}}
	//service1, _ := balancer(uint64(0))
	suite.Equal(*(serviceArray.Arr[0]), service)
}
func TestFilterTestSuite(t *testing.T) {
	suite.Run(t, new(FilterTestSuite))
}
