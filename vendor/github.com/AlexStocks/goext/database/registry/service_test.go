package gxregistry

import (
	"testing"
)

import (
	"github.com/stretchr/testify/suite"
)

type ServiceAddrTestSuite struct {
	suite.Suite
	sa   ServiceAttr
	node Node
}

func (suite *ServiceAddrTestSuite) SetupSuite() {
	suite.sa = ServiceAttr{
		Group:    "bjtelecom",
		Service:  "shopping",
		Protocol: "pb",
		Version:  "1.0.1",
		Role:     SRT_Provider,
	}

	suite.node = Node{ID: "node1", Address: "127.0.0.1", Port: 12345}
}

func (suite *ServiceAddrTestSuite) TearDownSuite() {
	suite.sa = ServiceAttr{}
}

func (suite *ServiceAddrTestSuite) TestServiceAttr_MarshalPath() {
	saBytes, err := suite.sa.MarshalPath()
	suite.T().Logf("sa string:%#v, err:%#v", string(saBytes), err)
	saStr := "group%3Dbjtelecom%26protocol%3Dpb%26role%3DSRT_Provider%26service%3Dshopping%26version%3D1.0.1"
	suite.Equalf([]byte(saStr), saBytes, "Marshal(sa:%+v)", suite.sa)
	suite.Equalf(nil, err, "Marshal(sa:%+v)", suite.sa)
}

func (suite *ServiceAddrTestSuite) TestServiceAttr_UnmarshalPath() {
	var sa ServiceAttr
	saStr := "group%3Dbjtelecom%26protocol%3Dpb%26role%3DSRT_Provider%26service%3Dshopping%26version%3D1.0.1"
	err := (&sa).UnmarshalPath([]byte(saStr))
	suite.T().Logf("suite.sa:%+v, sa:%+v", suite.sa, sa)
	suite.Equalf(sa, suite.sa, "Unmarshal(sa:%+v)", suite.sa)
	suite.Equalf(nil, err, "Unmarshal(sa:%+v)", suite.sa)
}

// Path example: /dubbo/shopping-bjtelecom-pb-1.0.1/node1
func (suite *ServiceAddrTestSuite) TestService_NodePath() {
	service := Service{
		Attr:  &suite.sa,
		Nodes: []*Node{&suite.node},
	}
	path := service.Path("/dubbo")
	saStr := "group%3Dbjtelecom%26protocol%3Dpb%26role%3DSRT_Provider%26service%3Dshopping%26version%3D1.0.1"
	suite.Equalf("/dubbo/"+saStr+"/", path, "service:%+v, path:%s", service, path)
	suite.T().Logf("service path:%s", path)

	path = service.NodePath("/dubbo", suite.node)
	suite.Equalf("/dubbo/"+saStr+"/node1", path, "service:%+v, path:%s", service, path)
	suite.T().Logf("node path:%s", path)
}

func TestServiceAddrTestSuite(t *testing.T) {
	suite.Run(t, new(ServiceAddrTestSuite))
}
