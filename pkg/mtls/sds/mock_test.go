/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package sds

import (
	"context"
	"errors"
	"time"

	"mosn.io/mosn/pkg/types"
)

type MockSdsStreamClient struct {
	ch      chan string
	err     error
	timer   *time.Timer
	timeout time.Duration
	closed  bool
}

type MockSdsConfig struct {
	Timeout  time.Duration
	ErrorStr string
}

func NewMockSdsStreamClient(config interface{}) (SdsStreamClient, error) {
	cfg, ok := config.(*MockSdsConfig)
	if !ok {
		return nil, errors.New("invalid config")
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = time.Hour // never timeout
	}
	var err error
	if cfg.ErrorStr != "" {
		err = errors.New(cfg.ErrorStr)
	}
	return &MockSdsStreamClient{
		ch:      make(chan string, 1),
		err:     err,
		timer:   time.NewTimer(cfg.Timeout),
		timeout: cfg.Timeout,
		closed:  false,
	}, nil

}

func (msc *MockSdsStreamClient) Send(name string) error {
	if msc.err != nil {
		return msc.err
	}
	if msc.closed {
		return errors.New("client closed")
	}
	select {
	case msc.ch <- name:
		return nil
	default:
		return errors.New("send failed")
	}
}

func (msc *MockSdsStreamClient) Recv(provider types.SecretProvider, callback func()) error {
	if msc.err != nil {
		return msc.err
	}
	if msc.closed {
		return errors.New("client closed")
	}
	msc.timer.Reset(msc.timeout)
	select {
	case name, ok := <-msc.ch:
		if ok {
			provider.SetSecret(name, &types.SdsSecret{
				Name: name,
			})
			if callback != nil {
				callback()
			}
		}
	case <-msc.timer.C:
		return errors.New("receive timeout")
	}
	return nil
}

func (msc *MockSdsStreamClient) Fetch(ctx context.Context, name string) (*types.SdsSecret, error) {
	return nil, errors.New("not implement yet")
}

func (msc *MockSdsStreamClient) AckResponse(resp interface{}) {}

func (msc *MockSdsStreamClient) Stop() {
	close(msc.ch)
	msc.closed = true
}
