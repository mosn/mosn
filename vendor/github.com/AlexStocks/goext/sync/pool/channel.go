// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

package gxpool

import (
	"sync"
)

import (
	"github.com/juju/errors"
)

// channelPool implements the Pool interface based on buffered channels.
type channelPool struct {
	// storage for our Object Objects
	mu   sync.Mutex
	pool chan Object
}

// NewChannelPool returns a new pool based on buffered channels maximum capacity.
// Pool doesn't fill the Pool until the (Pool)Put() is called. If there is no new Object
// available in the pool, (Pool)Get will return nil and u should create a new one
// by yourself.
func NewChannelPool(poolLen int) (Pool, error) {
	if poolLen <= 0 {
		return nil, errors.New("invalid capacity settings")
	}

	return &channelPool{pool: make(chan Object, poolLen)}, nil
}

func (c *channelPool) getPool() chan Object {
	c.mu.Lock()
	pool := c.pool
	c.mu.Unlock()
	return pool
}

// Get implements the Pool interfaces Get() method. If there is no new
// Object available in the pool, it will return nil.
func (c *channelPool) Get() (Object, error) {
	pool := c.getPool()
	if pool == nil {
		return nil, ErrClosed
	}

	// puts the Object back to the pool if it's closed.
	select {
	case obj := <-pool:
		if obj == nil {
			return nil, ErrClosed
		}

		return obj, nil
	default:
		return nil, ErrEmpty
	}
}

// put puts the Object back to the pool. If the pool is full or closed,
// @obj is simply closed. A nil obj will be rejected.
func (c *channelPool) Put(obj Object) error {
	if obj == nil {
		return errors.New("@obj is nil. rejecting")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.pool == nil {
		// pool is closed, close passed Object
		return obj.Close()
	}

	// put the resource back into the pool. If the pool is full, this will
	// block and the default case will be executed.
	select {
	case c.pool <- obj:
		return nil
	default:
		// pool is full, close passed Object
		return obj.Close()
	}
}

func (c *channelPool) Close() {
	c.mu.Lock()
	pool := c.pool
	c.pool = nil
	c.mu.Unlock()

	if pool == nil {
		return
	}

	close(pool)
	for obj := range pool {
		obj.Close()
	}
}

func (c *channelPool) Len() int {
	return len(c.getPool())
}
