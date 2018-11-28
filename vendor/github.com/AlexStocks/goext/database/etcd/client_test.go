package gxetcd

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/AlexStocks/goext/database/registry"
	etcdv3 "github.com/coreos/etcd/clientv3"
	jerrors "github.com/juju/errors"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	config etcdv3.Config
	client *Client
	wg     sync.WaitGroup
}

func (suite *ClientTestSuite) SetupSuite() {
	suite.config = etcdv3.Config{
		Endpoints:            []string{"127.0.0.1:2379"},
		DialTimeout:          8e9,
		DialKeepAliveTimeout: 3e9,
	}
}

func (suite *ClientTestSuite) SetupTest() {
	etcdClient, err := etcdv3.New(suite.config)
	if err != nil {
		panic(jerrors.Errorf("etcdv3.New(config:%+v) = error:%s", suite.config, err))
	}
	suite.client, _ = NewClient(etcdClient, WithTTL(7e9))
}

func (suite *ClientTestSuite) TearDownTest() {
	suite.client.Close()
}

func (suite *ClientTestSuite) TearDownSuite() {
	suite.client.Close()
	suite.wg.Wait()
}

func (suite *ClientTestSuite) TestClient_TTL() {
	suite.T().Logf("ttl:%s", time.Duration(suite.client.TTL()))
}

func (suite *ClientTestSuite) TestClient_Keepalive() {
	// suite.T().Logf("start to test keep alvie")
	fmt.Println("start to test keep alvie")
	keepAlive, err := suite.client.KeepAlive()
	suite.Equal(nil, err, "etcd.KeepAlive()")
	suite.wg.Add(1)
	go func() {
		var failTime int
		defer suite.wg.Done()
		failTime = 0
		for {
			select {
			case <-suite.client.Done():
				fmt.Println("keep alive goroutine exit now ...")
				return
			case msg, ok := <-keepAlive:
				// eat messages until keep alive channel closes
				if !ok {
					fmt.Println("keep alive channel closed")
					keepAlive, err = suite.client.KeepAlive()
					suite.Equal(nil, err, "etcd.KeepAlive()")
					failTime <<= 1
					if failTime == 0 {
						failTime = 1e8
					} else if gxregistry.MaxFailTime < failTime {
						failTime = gxregistry.MaxFailTime
					}
					fmt.Printf("%d, sleep time:%s\n", failTime/1e8, time.Duration(failTime))
					time.Sleep(time.Duration(failTime)) // to avoid connecting the registry tool frequently
				} else {
					failTime = 0
					fmt.Printf("Recv msg from keepAlive: %s\n", msg.String())
				}
			case <-time.After(2e9):
				fmt.Println("tick tock ...")
			}
		}
	}()

	time.Sleep(120e9)
	// suite.T().Logf("finish testing keep alvie")
	fmt.Println("finish testing keep alvie")
}

func (suite *ClientTestSuite) TestClient_Close() {
	suite.client.Stop()
	err := suite.client.Close()
	suite.Equal(nil, err, "Client.Close() = error:%s", jerrors.ErrorStack(err))
	flag := suite.client.IsClosed()
	suite.Equal(true, flag)
	err = suite.client.Close()
	suite.Equal(nil, err, "Client.Close() = error:%s", jerrors.ErrorStack(err))
	flag = suite.client.IsClosed()
	suite.Equal(true, flag)
	// suite.T().Logf("Client.Close() = error:%s", jerrors.ErrorStack(err))
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}
