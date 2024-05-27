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

package router

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/dchest/siphash"
	"mosn.io/api"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/variable"
)

// [sub module] & [function] & msg
const RouterLogFormat = "[router] [%s] [%s] %+v"

var (
	ErrNilRouterConfig      = errors.New("router config is nil")
	ErrNoVirtualHost        = errors.New("virtual host is nil")
	ErrNoRouters            = errors.New("routers is nil")
	ErrDuplicateVirtualHost = errors.New("duplicate domain virtual host")
	ErrDuplicateHostPort    = errors.New("duplicate virtual host port")
	ErrNoVirtualHostPort    = errors.New("virtual host port is invalid")
	ErrUnexpected           = errors.New("an unexpected error occurs")
)

var defaultRouteHandlerName = types.DefaultRouteHandler

func GetDefaultRouteHandlerName() string {
	return defaultRouteHandlerName
}

type headerFormatter interface {
	format(ctx context.Context) string
	append() bool
}

type headerPair struct {
	headerName      string // should to be lower string
	headerFormatter headerFormatter
}

type weightedClusterEntry struct {
	clusterName                  string
	clusterWeight                uint32
	clusterMetadataMatchCriteria *MetadataMatchCriteriaImpl
}

type RouteBase = api.RouteBase

// Policy
type policy struct {
	retryPolicy  *retryPolicyImpl
	shadowPolicy *shadowPolicyImpl //TODO: not implement yet
	hashPolicy   api.HashPolicy
	mirrorPolicy api.MirrorPolicy
}

func (p *policy) RetryPolicy() api.RetryPolicy {
	return p.retryPolicy
}

func (p *policy) ShadowPolicy() api.ShadowPolicy {
	return p.shadowPolicy
}

func (p *policy) HashPolicy() api.HashPolicy {
	return p.hashPolicy
}

func (p *policy) MirrorPolicy() api.MirrorPolicy {
	return p.mirrorPolicy
}

type retryPolicyImpl struct {
	retryOn      bool
	retryTimeout time.Duration
	numRetries   uint32
	statusCodes  []uint32
}

func (p *retryPolicyImpl) RetryOn() bool {
	if p == nil {
		return false
	}
	return p.retryOn
}

func (p *retryPolicyImpl) TryTimeout() time.Duration {
	if p == nil {
		return 0
	}
	return p.retryTimeout
}

func (p *retryPolicyImpl) NumRetries() uint32 {
	if p == nil {
		return 0
	}
	return p.numRetries
}

func (p *retryPolicyImpl) RetryableStatusCodes() []uint32 {
	if p == nil {
		return []uint32{}
	}
	return p.statusCodes
}

type shadowPolicyImpl struct {
	cluster    string
	runtimeKey string
}

func (spi *shadowPolicyImpl) ClusterName() string {
	return spi.cluster
}

func (spi *shadowPolicyImpl) RuntimeKey() string {
	return spi.runtimeKey
}

// RouterRuleFactory creates a RouteBase
type RouterRuleFactory func(base *RouteRuleImplBase, header []v2.HeaderMatcher) RouteBase

// The register order, is a wrapper of registered factory
// We register a factory with order, a new factory can replace old registered factory only if the register order
// ig greater than the old one.
type routerRuleFactoryOrder struct {
	factory RouterRuleFactory
	order   uint32
}

// if name is matched failed, use default factory
type handlerFactories struct {
	mutex          sync.RWMutex
	factories      map[string]MakeHandlerFunc
	defaultFactory MakeHandlerFunc
}

func (f *handlerFactories) add(name string, h MakeHandlerFunc, isDefault bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.factories[name] = h
	if isDefault {
		defaultRouteHandlerName = name
		f.defaultFactory = h
	}
}

func (f *handlerFactories) get(name string) MakeHandlerFunc {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	if h, ok := f.factories[name]; ok {
		return h
	}
	return f.defaultFactory
}

func (f *handlerFactories) exists(name string) bool {
	f.mutex.RLock()
	defer f.mutex.RUnlock()
	_, ok := f.factories[name]
	return ok
}

type headerHashPolicyImpl struct {
	key string
}

func (hp *headerHashPolicyImpl) GenerateHash(ctx context.Context) uint64 {
	headerKey := hp.key
	headerValue, err := variable.GetProtocolResource(ctx, api.HEADER, headerKey)

	if err == nil {
		hashString := fmt.Sprintf("%s:%s", headerKey, headerValue)
		hash := getHashByString(hashString)
		return hash
	}
	return 0
}

type cookieHashPolicyImpl struct {
	name string
	// path and ttl field are used for generate cookie value,
	// they are not being used currently.
	path string
	ttl  api.DurationConfig
}

// GenerateHash is httpCookieHashPolicyImpl hash generate logic.
//
// !!! please notice, in envoy or istio cookie may be generated if cookie value is not found,
// MOSN does NOT implement this strategy yet. When cookie value is not found a
// hash '0' will always be returned.
func (hp *cookieHashPolicyImpl) GenerateHash(ctx context.Context) uint64 {
	cookieName := hp.name
	cookieValue, err := variable.GetProtocolResource(ctx, api.COOKIE, cookieName)
	if err == nil {
		h := getHashByString(fmt.Sprintf("%s=%s", cookieName, cookieValue))
		return h
	}
	return 0
}

type sourceIPHashPolicyImpl struct{}

func (hp *sourceIPHashPolicyImpl) GenerateHash(ctx context.Context) uint64 {
	if addrv, err := variable.Get(ctx, types.VariableOriRemoteAddr); err == nil {
		if addr, ok := addrv.(net.Addr); ok {
			return getHashByAddr(addr)
		}
	}
	return 0
}

func getHashByAddr(addr net.Addr) (hash uint64) {
	if tcpaddr, ok := addr.(*net.TCPAddr); ok {
		if len(tcpaddr.IP) == 16 || len(tcpaddr.IP) == 4 {
			var tmp uint32

			if len(tcpaddr.IP) == 16 {
				tmp = binary.BigEndian.Uint32(tcpaddr.IP[12:16])
			} else {
				tmp = binary.BigEndian.Uint32(tcpaddr.IP)
			}
			hash = uint64(tmp)

			return
		}
	}

	return getHashByString(fmt.Sprintf("%s", addr.String()))
}

func getHashByString(str string) uint64 {
	return siphash.Hash(0xbeefcafebabedead, 0, []byte(str))
}

type mirrorImpl struct {
	cluster string
	percent int
	rand    *rand.Rand
}

func (m *mirrorImpl) IsMirror() (isTrans bool) {
	if m.cluster == "" || m.percent == 0 {
		return false
	}
	return m.percent > m.rand.Intn(100)
}

func (m *mirrorImpl) ClusterName() string {
	return m.cluster
}
