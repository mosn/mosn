// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxetcd provides an etcd version 3 gxregistry
// ref: https://github.com/micro/go-plugins/blob/master/gxregistry/etcdv3/etcdv3.go
package gxetcd

import (
	"context"
	"strings"
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
	"github.com/coreos/etcd/clientv3"
	jerrors "github.com/juju/errors"
)

import (
	etcd "github.com/AlexStocks/goext/database/etcd"
	"github.com/AlexStocks/goext/database/registry"
)

// watcher的watch系列函数暴露给registry，而Next函数则暴露给selector
type Watcher struct {
	done      chan struct{}
	cancel    context.CancelFunc
	w         clientv3.WatchChan
	opts      gxregistry.WatchOptions
	client    *etcd.Client
	sync.Once // for Close
}

func NewWatcher(client *etcd.Client, opts ...gxregistry.WatchOption) (gxregistry.Watcher, error) {
	var options gxregistry.WatchOptions
	for _, o := range opts {
		o(&options)
	}

	if options.Root == "" {
		options.Root = gxregistry.DefaultServiceRoot
	}

	if client.TTL() < 0 {
		// there is no lease
		// fix bug of TestValid, 20180421
		_, err := client.KeepAlive()
		if err != nil {
			return nil, jerrors.Trace(err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	watchPath := options.Root
	if !strings.HasSuffix(watchPath, "/") {
		watchPath += "/"
	}
	w := client.EtcdClient().Watch(ctx, watchPath, clientv3.WithPrefix(), clientv3.WithPrevKV())

	wc := &Watcher{
		done:   make(chan struct{}, 1),
		cancel: cancel,
		w:      w,
		opts:   options,
		client: client,
	}

	return wc, nil
}

func (w *Watcher) Notify() (*gxregistry.EventResult, error) {
	var (
		err     error
		service *gxregistry.Service
		action  gxregistry.ServiceEventType
	)

	for msg := range w.w {
		if w.IsClosed() {
			return nil, gxregistry.ErrWatcherClosed
		}

		if msg.Err() != nil {
			return nil, msg.Err()
		}

		for _, ev := range msg.Events {
			switch ev.Type {
			case clientv3.EventTypePut:
				if ev.IsCreate() {
					action = gxregistry.ServiceAdd
				} else if ev.IsModify() {
					action = gxregistry.ServiceUpdate
				}

				service, err = gxregistry.DecodeService(ev.Kv.Value)
				if err != nil || service == nil {
					log.Warn("gxregistry.DecodeService() = {service:%p, error:%+v}", service, jerrors.ErrorStack(err))
					continue
				}

			case clientv3.EventTypeDelete:
				action = gxregistry.ServiceDel

				// get service from prevKv
				service, err = gxregistry.DecodeService(ev.PrevKv.Value)
				if err != nil || service == nil {
					log.Warn("gxregistry.DecodeService() = {service:%p, error:%+v}", service, jerrors.ErrorStack(err))
					continue
				}
			}

			return &gxregistry.EventResult{
				Action:  action,
				Service: service,
			}, nil
		}
	}

	return nil, jerrors.Errorf("could not get next")
}

func (w *Watcher) Valid() bool {
	if w.IsClosed() {
		return false
	}

	return w.client.TTL() > 0
}

func (w *Watcher) Close() {
	w.Once.Do(func() {
		select {
		case <-w.done:
			return
		default:
			close(w.done)
			w.cancel()
		}
	})
}

// check whether the session has been closed.
func (w *Watcher) IsClosed() bool {
	select {
	case <-w.done:
		return true

	default:
		return false
	}
}
