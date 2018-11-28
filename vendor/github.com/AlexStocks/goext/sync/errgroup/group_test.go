package gxerrgroup

import (
	"errors"
	"time"
)

import (
	"github.com/stretchr/testify/suite"
	"testing"
)

type GroupTestSuite struct {
	suite.Suite
	g *Group
}

func (suite *GroupTestSuite) SetupSuite() {
}

func (suite *GroupTestSuite) TearDownSuite() {
}

func (suite *GroupTestSuite) SetupTest() {
	suite.g = NewGroup(60e9)
}

func (suite *GroupTestSuite) TearDownTest() {
	suite.g.Close()
}

func (suite *GroupTestSuite) TestZero() {
	res := make(chan error)
	go func() { res <- suite.g.Run() }()
	select {
	case err := <-res:
		if err != nil {
			suite.Error(err)
		}
	case <-time.After(100 * time.Millisecond):
		suite.Error(nil, "timeout")
	}
}

func (suite *GroupTestSuite) TestOne() {
	myError := errors.New("foobar")
	suite.g.Add(func() error { return myError }, func(error) {})
	res := make(chan error)
	go func() { res <- suite.g.Run() }()
	select {
	case err := <-res:
		if want, have := myError, err; want != have {
			suite.Errorf(err, "want %v, have %v", want, have)
		}
	case <-time.After(100 * time.Millisecond):
		suite.Error(nil, "timeout")
	}
}

func (suite *GroupTestSuite) TestMany() {
	interrupt := errors.New("interrupt")
	suite.g.Add(func() error { return interrupt }, func(error) {})
	cancel := make(chan struct{})
	suite.g.Add(func() error { <-cancel; return nil }, func(error) { close(cancel) })
	res := make(chan error)
	go func() { res <- suite.g.Run() }()
	select {
	case err := <-res:
		if want, have := interrupt, err; want != have {
			suite.Errorf(err, "want %v, have %v", want, have)
		}
	case <-time.After(100 * time.Millisecond):
		suite.Error(nil, "timeout")
	}
}

func TestGroupTestSuite(t *testing.T) {
	suite.Run(t, new(GroupTestSuite))
}
