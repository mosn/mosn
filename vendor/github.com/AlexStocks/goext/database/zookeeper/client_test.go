package gxzookeeper

import (
	// "fmt"
	"strings"
	"testing"
	// "time"
)

import (
	// jerrors "github.com/juju/errors"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	client *Client
}

func (suite *ClientTestSuite) SetupSuite() {
}

func (suite *ClientTestSuite) SetupTest() {
	conn, _, _ := zk.Connect([]string{"127.0.0.1:2181"}, 3e9)
	suite.client = NewClient(conn)
}

func (suite *ClientTestSuite) TearDownTest() {
	suite.client.ZkConn().Close()
}

func (suite *ClientTestSuite) TearDownSuite() {
}

func (suite *ClientTestSuite) TestClient_RegisterTempSeq() {
	path := "/test"
	err := suite.client.CreateZkPath(path)
	suite.Equal(nil, err, "CreateZkPath")

	path += "/hello"
	data := "world"
	tempPath, err := suite.client.RegisterTempSeq(path, []byte(data))
	suite.Equal(nil, err, "RegisterTempSeq")
	suite.Equal(true, strings.HasPrefix(tempPath, path), "tempPath:%s", tempPath)

	suite.client.DeleteZkPath(path)
}

func (suite *ClientTestSuite) TestClient_RegisterTemp() {
	path := "/getty-root"
	err := suite.client.CreateZkPath(path)
	suite.Equal(nil, err, "CreateZkPath")

	path += "/group%3Dbj-yizhuang%26protocol%3Dprotobuf%26role%3DSRT_Provider%26service" +
		"%3DTestService%26version%3Dv1.0"
	err = suite.client.CreateZkPath(path)
	suite.Equal(nil, err, "CreateZkPath")

	path += "/svr-node1"
	data := "world"
	tempPath, err := suite.client.RegisterTemp(path, []byte(data))
	suite.Equal(nil, err, "RegisterTemp")
	suite.Equal(true, strings.HasPrefix(tempPath, path), "tempPath:%s", tempPath)
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
