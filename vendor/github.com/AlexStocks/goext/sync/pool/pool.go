// Copyright 2016 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed Apache License 2.0.

// Package pool implements a pool of Object interfaces to manage and reuse them.
package gxpool

import (
	"github.com/juju/errors"
)

var (
	// ErrClosed is the error resulting if the pool is closed via pool.Close().
	ErrEmpty  = errors.New("pool is empty")
	ErrClosed = errors.New("pool is closed")
)

// Object is Pool object
type Object interface {
	Close() error
}

// Pool interface describes a pool implementation. A pool should have maximum
// capacity. An ideal pool is threadsafe and easy to use.
type Pool interface {
	// Get returns a new connection from the pool. Closing the connections puts
	// it back to the Pool. Closing it when the pool is destroyed or full will
	// be counted as an error.
	Get() (Object, error)

	Put(Object) error

	// Close closes the pool and all its connections. After Close() the pool is
	// no longer usable.
	Close()

	// Len returns the current number of connections of the pool.
	Len() int
}
