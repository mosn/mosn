// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.
//
// gxetcd encapsulate a etcd lease client
package gxetcd

import (
	"context"
	"time"
)

import (
	ecv3 "github.com/coreos/etcd/clientv3"
)

const (
	defaultClientTTL = 60e9
)

type clientOptions struct {
	ttl     time.Duration
	leaseID ecv3.LeaseID
	ctx     context.Context
}

//////////////////////////////////////////
// Lease Client Option
//////////////////////////////////////////

// ClientOption configures Client.
type ClientOption func(*clientOptions)

// WithTTL configures the session's TTL in seconds.
// If TTL is <= 0, the default 60 seconds TTL will be used.
func WithTTL(ttl time.Duration) ClientOption {
	return func(so *clientOptions) {
		if ttl > 0 {
			so.ttl = ttl
		}
	}
}

// WithLease specifies the existing leaseID to be used for the session.
// This is useful in process restart scenario, for example, to reclaim
// leadership from an election prior to restart.
func WithLease(leaseID ecv3.LeaseID) ClientOption {
	return func(so *clientOptions) {
		so.leaseID = leaseID
	}
}

// WithContext assigns a context to the session instead of defaulting to
// using the client context. This is useful for canceling NewClient and
// Close operations immediately without having to close the client. If the
// context is canceled before Close() completes, the session's lease will be
// abandoned and left to expire instead of being revoked.
func WithContext(ctx context.Context) ClientOption {
	return func(so *clientOptions) {
		so.ctx = ctx
	}
}
