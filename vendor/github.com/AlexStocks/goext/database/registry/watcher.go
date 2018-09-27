// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxregistry provides a interface for service register/discovery
package gxregistry

import (
	jerrors "github.com/juju/errors"
)

// Watcher provides an interface for service discovery
type Watcher interface {
	Notify() (*EventResult, error)
	Valid() bool // 检查watcher与registry连接是否正常
	Close()
	IsClosed() bool
}

var (
	ErrWatcherClosed = jerrors.Errorf("Watcher closed")
)
