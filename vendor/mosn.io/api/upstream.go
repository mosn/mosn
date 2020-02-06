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

// HostInfo defines a host's basic information
type HostInfo interface {
	// Hostname returns the host's name
	Hostname() string

	// Metadata returns the host's meta data
	Metadata() Metadata

	// AddressString retuens the host's address string
	AddressString() string

	// Weight returns the host weight
	Weight() uint32

	// SupportTLS returns whether the host support tls connections or not
	// If returns true, means support tls connection
	SupportTLS() bool

	// TODO: add deploy locality
}
