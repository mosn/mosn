// Copyright 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of c source code is
// governed by Apache License 2.0.

// Package gxzookeeper provides a zookeeper driver based on samuel/go-zookeeper/zk
package gxzookeeper

import (
	"path"
	"strings"
)

import (
	jerrors "github.com/juju/errors"
	"github.com/samuel/go-zookeeper/zk"
)

type Client struct {
	conn *zk.Conn // 这个conn不能被close两次，否则会收到 “panic: close of closed channel”
}

func NewClient(conn *zk.Conn) *Client {
	return &Client{conn: conn}
}

func (c *Client) ZkConn() *zk.Conn {
	return c.conn
}

func (c *Client) StateToString(state zk.State) string {
	switch state {
	case zk.StateDisconnected:
		return "zookeeper disconnected"
	case zk.StateConnecting:
		return "zookeeper connecting"
	case zk.StateAuthFailed:
		return "zookeeper auth failed"
	case zk.StateConnectedReadOnly:
		return "zookeeper connect readonly"
	case zk.StateSaslAuthenticated:
		return "zookeeper sasl authenticaed"
	case zk.StateExpired:
		return "zookeeper connection expired"
	case zk.StateConnected:
		return "zookeeper conneced"
	case zk.StateHasSession:
		return "zookeeper has session"
	case zk.StateUnknown:
		return "zookeeper unknown state"
	case zk.State(zk.EventNodeDeleted):
		return "zookeeper node deleted"
	case zk.State(zk.EventNodeDataChanged):
		return "zookeeper node data changed"
	default:
		return state.String()
	}

	return "zookeeper unknown state"
}

// 节点须逐级创建
func (c *Client) CreateZkPath(basePath string) error {
	var (
		err     error
		tmpPath string
	)

	if strings.HasSuffix(basePath, "/") {
		basePath = strings.TrimSuffix(basePath, "/")
	}

	for _, str := range strings.Split(basePath, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		_, err = c.conn.Create(tmpPath, []byte(""), 0, zk.WorldACL(zk.PermAll))
		if err != nil {
			if err != zk.ErrNodeExists {
				return jerrors.Annotatef(err, "zk.Create(path:%s)", tmpPath)
			}
		}
	}

	return nil
}

// 像创建一样，删除节点的时候也只能从叶子节点逐级回退删除
// 当节点还有子节点的时候，删除是不会成功的
func (c *Client) DeleteZkPath(path string) error {
	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	err := c.conn.Delete(path, -1)
	if err != nil {
		return jerrors.Annotatef(err, "zk.Delete(path:%s)", path)
	}

	return nil
}

func (c *Client) RegisterTemp(path string, data []byte) (string, error) {
	var (
		err     error
		tmpPath string
	)

	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	tmpPath, err = c.conn.Create(path, data, zk.FlagEphemeral, zk.WorldACL(zk.PermAll))
	if err != nil {
		return "", jerrors.Annotatef(err, "zk.Create(%s, ephemeral)", path)
	}

	return tmpPath, nil
}

func (c *Client) RegisterTempSeq(path string, data []byte) (string, error) {
	var (
		err     error
		tmpPath string
	)

	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	tmpPath, err = c.conn.Create(path, data, zk.FlagEphemeral|zk.FlagSequence, zk.WorldACL(zk.PermAll))
	if err != nil {
		return "", jerrors.Annotatef(err, "zk.Create(%s, sequence | ephemeral)", path)
	}

	return tmpPath, nil
}

func (c *Client) GetChildrenW(path string) ([]string, <-chan zk.Event, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
		watch    <-chan zk.Event
	)

	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	children, stat, watch, err = c.conn.ChildrenW(path)
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, nil, jerrors.Errorf("path{%s} has none children", path)
		}
		return nil, nil, jerrors.Annotatef(err, "zk.ChildrenW(%s)", path)
	}
	if stat == nil {
		return nil, nil, jerrors.Errorf("path{%s} has none children", path)
	}
	if len(children) == 0 {
		return nil, nil, jerrors.Errorf("path{%s} has none children", path)
	}

	return children, watch, nil
}

func (c *Client) Get(path string) ([]byte, error) {
	var (
		err  error
		data []byte
		stat *zk.Stat
	)

	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	data, stat, err = c.conn.Get(path)
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, jerrors.Errorf("path{%s} has none children", path)
		}
		return nil, jerrors.Annotatef(err, "zk.Children(path:%s)", path)
	}
	if stat == nil {
		return nil, jerrors.Errorf("path{%s} has none children", path)
	}
	if len(data) == 0 {
		return nil, jerrors.Errorf("path{%s} has none children", path)
	}

	return data, nil
}

func (c *Client) GetChildren(path string) ([]string, error) {
	var (
		err      error
		children []string
		stat     *zk.Stat
	)

	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	children, stat, err = c.conn.Children(path)
	if err != nil {
		if err == zk.ErrNoNode {
			return nil, jerrors.Errorf("path{%s} has none children", path)
		}
		return nil, jerrors.Annotatef(err, "zk.Children(path:%s)", path)
	}
	if stat == nil || stat.NumChildren == 0 {
		return nil, jerrors.Errorf("path{%s} has none children", path)
	}
	if len(children) == 0 {
		return nil, jerrors.Errorf("path{%s} has none children", path)
	}

	return children, nil
}

func (c *Client) ExistW(path string) (<-chan zk.Event, error) {
	var (
		exist bool
		err   error
		watch <-chan zk.Event
	)

	if strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}

	exist, _, watch, err = c.conn.ExistsW(path)
	if err != nil {
		return nil, jerrors.Annotatef(err, "zk.ExistsW(path:%s)", path)
	}
	if !exist {
		return nil, jerrors.Errorf("zkClient App zk path{%s} does not exist.", path)
	}

	return watch, nil
}
