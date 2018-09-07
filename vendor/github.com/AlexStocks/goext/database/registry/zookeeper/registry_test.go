package gxzookeeper

import (
	"testing"
	"time"
)

import (
	"github.com/AlexStocks/goext/database/registry"
	"github.com/AlexStocks/goext/log"
	"github.com/stretchr/testify/suite"
)

type RegisterTestSuite struct {
	suite.Suite
	reg  gxregistry.Registry
	sa   gxregistry.ServiceAttr
	node gxregistry.Node
}

func (suite *RegisterTestSuite) SetupSuite() {
	suite.sa = gxregistry.ServiceAttr{
		Group:    "bjtelecom",
		Service:  "shopping",
		Protocol: "pb",
		Version:  "1.0.1",
		Role:     gxregistry.SRT_Provider,
	}

	suite.node = gxregistry.Node{ID: "node0", Address: "127.0.0.1", Port: 12345}
}

func (suite *RegisterTestSuite) TearDownSuite() {
}

func (suite *RegisterTestSuite) SetupTest() {
	var err error
	suite.reg, err = NewRegistry(
		gxregistry.WithAddrs([]string{"127.0.0.1:2181"}...),
		gxregistry.WithTimeout(3e9),
		gxregistry.WithRoot("/test"),
	)
	suite.Equal(nil, err, "NewRegistry()")
}

func (suite *RegisterTestSuite) TearDownTest() {
	suite.reg.Close()
}

func (suite *RegisterTestSuite) TestRegistry_Options() {
	opts := suite.reg.Options()
	suite.Equalf("/test", opts.Root, "reg.Options.Root")
	// suite.Equalf([]string{"127.0.0.1:2379", "127.0.0.1:12379", "127.0.0.1:22379"}, opts.Addrs, "reg.Options.Addrs")
	suite.Equalf([]string{"127.0.0.1:2181"}, opts.Addrs, "reg.Options.Addrs")
	suite.Equalf(time.Duration(3e9), opts.Timeout, "reg.Options.Timeout")
}

func (suite *RegisterTestSuite) TestRegistry_Register() {
	// register suite.node
	service := gxregistry.Service{Attr: &suite.sa, Nodes: []*gxregistry.Node{&suite.node}}
	err := suite.reg.Register(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)
	err = suite.reg.Register(service)
	suite.Equalf(gxregistry.ErrorAlreadyRegister, err, "Register(service:%+v)", service)
	_, flag := suite.reg.(*Registry).exist(service)
	suite.Equalf(true, flag, "Registry.exist(service:%s)", gxlog.PrettyString(service))

	// register node1
	node1 := gxregistry.Node{ID: "node1", Address: "127.0.0.1", Port: 12346}
	service = gxregistry.Service{Attr: &suite.sa, Nodes: []*gxregistry.Node{&node1}}
	err = suite.reg.Register(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)
	err = suite.reg.Register(service)
	suite.Equalf(gxregistry.ErrorAlreadyRegister, err, "Register(service:%+v)", service)
	_, flag = suite.reg.(*Registry).exist(service)
	suite.Equalf(true, flag, "Registry.exist(service:%s)", gxlog.PrettyString(service))

	service1, err := suite.reg.GetServices(suite.sa)
	suite.Equalf(nil, err, "GetService(ServiceAttr:%#v)", suite.sa)
	suite.Equalf(2, len(service1), "GetService(ServiceAttr:%+v)", suite.sa)
	_, flag = suite.reg.(*Registry).exist(service1[0])
	suite.Equalf(true, flag, "Registry.exist(service:%s)", gxlog.PrettyString(service1))

	// unregister node1
	service = gxregistry.Service{Attr: &suite.sa, Nodes: []*gxregistry.Node{&node1}}
	err = suite.reg.Deregister(service)
	suite.Equalf(nil, err, "Deregister(service:%+v)", service)
	_, flag = suite.reg.(*Registry).exist(service)
	suite.Equalf(false, flag, "Registry.exist(service:%s)", gxlog.PrettyString(service))

	service1, err = suite.reg.GetServices(suite.sa)
	suite.T().Log("services:", gxlog.ColorSprintln(service1))
	suite.Equalf(nil, err, "GetService(ServiceAttr:%#v)", suite.sa)
	suite.Equalf(1, len(service1), "GetService(ServiceAttr:%+v)", suite.sa)

	// unregister node0
	service = gxregistry.Service{Attr: &suite.sa, Nodes: []*gxregistry.Node{&suite.node}}
	err = suite.reg.Deregister(service)
	suite.Equalf(nil, err, "Deregister(service:%+v)", service)
	_, flag = suite.reg.(*Registry).exist(service)
	suite.Equalf(false, flag, "Registry.exist(service:%s)", gxlog.PrettyString(service))

	service1, err = suite.reg.GetServices(suite.sa)
	suite.T().Log("services:", gxlog.ColorSprintln(service1))
	// suite.Equalf(gxregistry.ErrorRegistryNotFound, err, "GetService(ServiceAttr:%#v)", suite.sa)
	suite.NotEqualf(nil, err, "GetService(ServiceAttr:%#v)", suite.sa)
}

func (suite *RegisterTestSuite) TestRegistry_ZookeeperRestart() {
	node1 := gxregistry.Node{ID: "node1", Address: "127.0.0.1", Port: 12346}
	service := gxregistry.Service{Attr: &suite.sa, Nodes: []*gxregistry.Node{&suite.node, &node1}}
	err := suite.reg.Register(service)
	suite.Equalf(nil, err, "Register(service:%+v)", service)

	time.Sleep(50e9)

	service1, err := suite.reg.GetServices(suite.sa)
	suite.Equalf(nil, err, "GetService(ServiceAttr:%#v)", suite.sa)
	suite.Equalf(2, len(service1), "GetService(ServiceAttr:%+v)", suite.sa)
}

func TestRegisterTestSuite(t *testing.T) {
	suite.Run(t, new(RegisterTestSuite))
}
