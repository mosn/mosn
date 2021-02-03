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

package api

import (
	"context"

	"mosn.io/pkg/buffer"
)

// FilterStatus type
type FilterStatus string

// FilterStatus types
const (
	Continue FilterStatus = "Continue"
	Stop     FilterStatus = "Stop"
)

// FilterManager is a groups of filters
type FilterManager interface {
	// AddReadFilter adds a read filter
	AddReadFilter(rf ReadFilter)

	// AddWriteFilter adds a write filter
	AddWriteFilter(wf WriteFilter)

	// ListReadFilter returns the list of read filters
	ListReadFilter() []ReadFilter

	// ListWriteFilters returns the list of write filters
	ListWriteFilters() []WriteFilter

	// InitializeReadFilters initialize read filters
	InitializeReadFilters() bool

	// OnRead is called on data read
	OnRead()

	// OnWrite is called before data write
	OnWrite(buffer []buffer.IoBuffer) FilterStatus
}

// ReadFilter is a connection binary read filter, registered by FilterManager.AddReadFilter
type ReadFilter interface {
	// OnData is called everytime bytes is read from the connection
	OnData(buffer buffer.IoBuffer) FilterStatus

	// OnNewConnection is called on new connection is created
	OnNewConnection() FilterStatus

	// InitializeReadFilterCallbacks initials read filter callbacks. It used by init read filter
	InitializeReadFilterCallbacks(cb ReadFilterCallbacks)
}

// WriteFilter is a connection binary write filter, only called by conn accept loop
type WriteFilter interface {
	// OnWrite is called before data write to raw connection
	OnWrite(buffer []buffer.IoBuffer) FilterStatus
}

// ReadFilterCallbacks is called by read filter to talk to connection
type ReadFilterCallbacks interface {
	// Connection returns the connection triggered the callback
	Connection() Connection

	// ContinueReading filter iteration on filter stopped, next filter will be called with current read buffer
	ContinueReading()

	// UpstreamHost returns current selected upstream host.
	UpstreamHost() HostInfo

	// SetUpstreamHost set currently selected upstream host.
	SetUpstreamHost(upstreamHost HostInfo)
}

// NetWorkFilterChainFactoryCallbacks is a wrapper of FilterManager that called in NetworkFilterChainFactory
type NetWorkFilterChainFactoryCallbacks interface {
	AddReadFilter(rf ReadFilter)
	AddWriteFilter(wf WriteFilter)
}

// NetworkFilterChainFactory adds filter into NetWorkFilterChainFactoryCallbacks
type NetworkFilterChainFactory interface {
	CreateFilterChain(context context.Context, callbacks NetWorkFilterChainFactoryCallbacks)
}
