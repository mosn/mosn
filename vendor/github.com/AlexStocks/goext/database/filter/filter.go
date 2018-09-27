// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxfilter provides a interface for service filter
package gxfilter

import (
	"github.com/AlexStocks/goext/database/registry"
	jerrors "github.com/juju/errors"
)

// Filter used to get service nodes from registry.
type Filter interface {
	Options() Options
	GetService(gxregistry.ServiceAttr) ([]*gxregistry.Service, error)
	Filter(gxregistry.ServiceAttr) (*ServiceArray, error)
	CheckServiceAlive(gxregistry.ServiceAttr, *ServiceArray) bool
	Close() error
}

var (
	ErrNotFound              = jerrors.Errorf("not found")
	ErrNoneAvailable         = jerrors.Errorf("none available")
	ErrRunOutAllServiceNodes = jerrors.Errorf("has used out all provider nodes")
)
