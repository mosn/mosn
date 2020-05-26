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

import "sync/atomic"

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
