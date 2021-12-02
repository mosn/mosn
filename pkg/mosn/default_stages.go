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

package mosn

import (
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/featuregate"
)

// mosn not inited yet, using the local config here
func (m *Mosn) PreInitConfig(c *v2.MOSNConfig) {
	InitDefaultPath(c)
	InitDebugServe(c)
	InitializePidFile(c)
	InitializeTracing(c)
	InitializePlugin(c)
	InitializeWasm(c)
	InitializeThirdPartCodec(c)
	m.Config = c
}

// mosn alreday inited
func (m *Mosn) PreStart() {
	// start xds client
	_ = m.StartXdsClient()
	featuregate.FinallyInitFunc()
	m.HandleExtendConfig()
}

// transfer existing connections from old mosn,
// stage manager will stop the new mosn when return error
func (m *Mosn) InheritConnections() error {
	// transfer connection used in smooth upgrade in mosn
	err := m.TransferConnection()
	// clean upgrade finish the smooth upgrade datas
	m.CleanUpgrade()
	return err
}
