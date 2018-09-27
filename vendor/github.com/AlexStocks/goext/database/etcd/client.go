// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.
//
// gxetcd encapsulate a etcd lease client
package gxetcd

import (
	"context"
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
	ecv3 "github.com/coreos/etcd/clientv3"
	jerrors "github.com/juju/errors"
)

//////////////////////////////////////////
// Lease Client
//////////////////////////////////////////

// Client represents a lease kept alive for the lifetime of a client.
// Fault-tolerant applications may use sessions to reason about liveness.
type Client struct {
	client *ecv3.Client
	opts   *clientOptions
	done   chan struct{}
	sync.Mutex
	id     ecv3.LeaseID
	cancel context.CancelFunc
}

// NewClient gets the leased session for a client.
func NewClient(client *ecv3.Client, options ...ClientOption) (*Client, error) {
	opts := &clientOptions{ttl: defaultClientTTL, ctx: client.Ctx()}
	for _, opt := range options {
		opt(opts)
	}

	c := &Client{client: client, opts: opts, id: opts.leaseID, done: make(chan struct{})}
	log.Info("new etcd client %+v", c)

	return c, nil
}

func (c *Client) KeepAlive() (<-chan *ecv3.LeaseKeepAliveResponse, error) {
	c.Lock()
	if c.cancel != nil {
		c.cancel()
		c.cancel = nil
	}
	id := c.id
	c.Unlock()

	if id == ecv3.NoLease {
		rsp, err := c.client.Grant(c.opts.ctx, int64(c.opts.ttl.Seconds()))
		if err != nil {
			return nil, jerrors.Trace(err)
		}
		id = rsp.ID
	}
	c.Lock()
	c.id = id
	c.Unlock()

	ctx, cancel := context.WithCancel(c.opts.ctx)
	keepAlive, err := c.client.KeepAlive(ctx, id)
	if err != nil || keepAlive == nil {
		c.client.Revoke(ctx, id)
		c.id = ecv3.NoLease
		cancel()
		return nil, jerrors.Annotatef(err, "etcdv3.KeepAlive(id:%#X)", id)
	}

	c.Lock()
	c.cancel = cancel
	c.Unlock()

	return keepAlive, nil
}

// Client is the etcd client that is attached to the session.
func (c *Client) EtcdClient() *ecv3.Client {
	return c.client
}

// Lease is the lease ID for keys bound to the session.
func (c *Client) Lease() ecv3.LeaseID { return c.id }

// TTL return the ttl of Client's lease
func (c *Client) TTL() int64 {
	rsp, err := c.client.TimeToLive(context.TODO(), c.id)
	if err != nil {
		return 0
	}

	return rsp.TTL
}

// Done returns a channel that closes when the lease is orphaned, expires, or
// is otherwise no longer being refreshed.
func (c *Client) Done() <-chan struct{} { return c.done }

// check whether the session has been closed.
func (c *Client) IsClosed() bool {
	select {
	case <-c.done:
		return true

	default:
		return false
	}
}

// Stop ends the refresh for the session lease. This is useful
// in case the state of the client connection is indeterminate (revoke
// would fail) or when transferring lease ownership.
func (c *Client) Stop() {
	select {
	case <-c.done:
		return

	default:
		c.Lock()
		if c.cancel != nil {
			c.cancel()
			c.cancel = nil
		}
		close(c.done)
		c.Unlock()
		return
	}
}

// Close orphans the session and revokes the session lease.
func (c *Client) Close() error {
	c.Stop()
	var err error
	c.Lock()
	id := c.id
	c.id = ecv3.NoLease
	c.Unlock()
	if id != ecv3.NoLease {
		// if revoke takes longer than the ttl, lease is expired anyway
		ctx, cancel := context.WithTimeout(c.opts.ctx, c.opts.ttl)
		_, err = c.client.Revoke(ctx, id)
		cancel()
	}

	if err != nil {
		err = jerrors.Annotatef(err, "etcdv3.Remove(lieaseID:%+v)", id)
	}

	return err
}
