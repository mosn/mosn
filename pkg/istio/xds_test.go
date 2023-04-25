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

package istio

import (
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	v2 "mosn.io/mosn/pkg/config/v2"
)

type MockXdsStreamConfig struct {
	failed      uint32
	delay       time.Duration
	recvTimeout time.Duration
	handle      func(interface{})
}

func (sc *MockXdsStreamConfig) CreateXdsStreamClient() (XdsStreamClient, error) {
	if atomic.LoadUint32(&sc.failed) != 0 {
		// next time will be success
		atomic.StoreUint32(&sc.failed, 0)
		return nil, errors.New("get stream client error")
	}
	return &MockXdsStreamClient{
		ch:      make(chan string, 1),
		closed:  false,
		timeout: sc.recvTimeout,
		handle:  sc.handle,
	}, nil
}

func (sc *MockXdsStreamConfig) RefreshDelay() time.Duration {
	return sc.delay
}

func (sc *MockXdsStreamConfig) InitAdsRequest() interface{} {
	return "cds-test"
}

type MockXdsStreamClient struct {
	ch      chan string
	closed  bool
	timeout time.Duration
	handle  func(interface{})
}

func (sc *MockXdsStreamClient) Send(req interface{}) error {
	s, ok := req.(string)
	if !ok {
		return errors.New("invalid request")
	}
	if sc.closed {
		return errors.New("client closed")
	}
	sc.ch <- s
	return nil
}

func (sc *MockXdsStreamClient) Recv() (interface{}, error) {
	select {
	case s, ok := <-sc.ch:
		if !ok {
			return "", errors.New("client closed")
		}
		return s, nil
	case <-time.After(sc.timeout):
		return nil, errors.New("recv timeout")
	}
}

func (sc *MockXdsStreamClient) HandleResponse(resp interface{}) {
	sc.handle(resp)
}

func (sc *MockXdsStreamClient) Stop() {
	sc.closed = true
	close(sc.ch)
}

func waitStop(c *ADSClient) {
	c.Stop()
	<-c.stopChan
	time.Sleep(time.Second) // wait clean

}

func TestXdsClient(t *testing.T) {
	records := map[string]struct{}{}
	cfg := &MockXdsStreamConfig{
		delay:       200 * time.Millisecond,
		recvTimeout: time.Second,
		handle: func(resp interface{}) {
			s := resp.(string)
			records[s] = struct{}{}
		},
	}
	RegisterParseAdsConfig(func(_, _ json.RawMessage) (XdsStreamConfig, error) {
		return cfg, nil
	})
	t.Run("start a xds client", func(t *testing.T) {
		client, _ := NewAdsClient(&v2.MOSNConfig{})
		ch := make(chan struct{})
		go func() {
			client.Start()
			ch <- struct{}{}
		}()
		select {
		case <-ch:
			// start try connect success
			require.NotNil(t, client.streamClient)
		case <-time.After(time.Second):
			t.Fatalf("start timeout")
		}
		time.Sleep(time.Second)
		_, ok := records["cds-test"]
		require.True(t, ok)
		waitStop(client)
		require.Nil(t, client.streamClient)
	})

	t.Run("disable reconnect client", func(t *testing.T) {
		records = map[string]struct{}{}
		DisableReconnect()
		cfg.failed = 1
		client, _ := NewAdsClient(&v2.MOSNConfig{})
		client.Start()
		time.Sleep(2 * time.Second)
		_, ok := records["cds-test"]
		require.False(t, ok)
		waitStop(client)

	})

	t.Run("reconnect client", func(t *testing.T) {
		records = map[string]struct{}{}
		EnableReconnect()
		cfg.failed = 1
		client, _ := NewAdsClient(&v2.MOSNConfig{})
		client.Start()
		time.Sleep(3 * time.Second)
		_, ok := records["cds-test"]
		require.True(t, ok)
		waitStop(client)
	})

	t.Run("reconnect client once", func(t *testing.T) {
		records = map[string]struct{}{}
		EnableReconnect()
		cfg.failed = 1
		client, _ := NewAdsClient(&v2.MOSNConfig{})
		client.Start()
		client.ReconnectStreamClient()
		time.Sleep(3 * time.Second)
		_, ok := records["cds-test"]
		require.True(t, ok)
		waitStop(client)
	})
}
