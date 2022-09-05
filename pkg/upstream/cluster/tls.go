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

package cluster

import (
	"crypto/sha256"
	"sync/atomic"

	"mosn.io/mosn/pkg/types"
)

// temporary implement:
// control client side tls support without update config
var isDisableClientSideTLS uint32

func DisableClientSideTLS() {
	atomic.StoreUint32(&isDisableClientSideTLS, 1)
}

func EnableClientSideTLS() {
	atomic.StoreUint32(&isDisableClientSideTLS, 0)
}

// IsSupportTLS returns the client side is support tls or not
// default is support
func IsSupportTLS() bool {
	return atomic.LoadUint32(&isDisableClientSideTLS) == 0
}

// disableTLSHashValue represents the host config's tls_disable is setted
// we use this hash value to distinguish the difference between nil and tls_disable
// so we can find the config changed during the isDisableClientSideTLS.
// Usually, the hash value is created by tls certificate info, so we create a simple hash value
// to represent the tls_disable.
var disableTLSHashValue *types.HashValue

// clientSideDisableHashValue represents the isDisableClientSideTLS is setted
// if IsSupportTLS == false, returns this hash value.
var clientSideDisableHashValue *types.HashValue

func init() {
	disableTLSHashValue = types.NewHashValue([sha256.Size]byte{0x00, 0x01, 0x02, 0x03})
	clientSideDisableHashValue = types.NewHashValue([sha256.Size]byte{}) // all data are 0x00
}
