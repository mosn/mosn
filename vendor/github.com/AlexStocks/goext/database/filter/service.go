// Copyright 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxfilter provides a interface for service filter
package gxfilter

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

import (
	"github.com/AlexStocks/goext/database/registry"
	jerrors "github.com/juju/errors"
)

/////////////////////////////////////
// ServiceArray
/////////////////////////////////////

var (
	ErrServiceArrayEmpty = jerrors.New("ServiceArray empty")
)

// ServiceHash is a user service hash interface to select a service provider.
// @context is transfered from ServiceArray::Select
type ServiceHash func(context.Context, *ServiceArray) (*gxregistry.Service, error)

type ServiceArray struct {
	Arr    []*gxregistry.Service
	Active time.Time
	idx    int64 // the last selected service provider index
}

func NewServiceArray(Arr []*gxregistry.Service) *ServiceArray {
	return &ServiceArray{
		Arr:    Arr,
		Active: time.Now(),
	}
}

func (s ServiceArray) String() string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Active:%s, idx:%d, Arr len:%d, Arr:{", s.Active, s.idx, len(s.Arr)))
	for i := range s.Arr {
		builder.WriteString(fmt.Sprintf("%d:%s, ", i, s.Arr[i]))
	}
	builder.WriteString("}")

	return builder.String()
}

func (s *ServiceArray) Select(ctx context.Context, hash ServiceHash) (*gxregistry.Service, error) {
	arrSize := len(s.Arr)
	if arrSize == 0 {
		return nil, ErrServiceArrayEmpty
	}

	if hash != nil {
		return hash(ctx, s)
	}

	// default use roundrobin hash alg
	idx := atomic.AddInt64(&s.idx, 1) % int64(arrSize)

	return s.Arr[idx], nil
}

func (s *ServiceArray) Add(service *gxregistry.Service, ttl time.Duration) {
	s.Arr = append(s.Arr, service)
	s.Active = time.Now().Add(ttl)
}

func (s *ServiceArray) Del(service *gxregistry.Service, ttl time.Duration) {
	for i, svc := range s.Arr {
		if svc.Equal(service) {
			s.Arr = append(s.Arr[:i], s.Arr[i+1:]...)
			s.Active = time.Now().Add(ttl)
			// log.Debug("i:%d, new services:%+v", i, services)
			break
		}
	}
}
