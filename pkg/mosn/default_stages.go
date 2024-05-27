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
	admin "mosn.io/mosn/pkg/admin/server"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/featuregate"
	"mosn.io/mosn/pkg/stagemanager"
)

// Default Init Stage wrappers. if more initialize needs to extend.
// modify it in main function.
// before inherit config from old mosn.
func DefaultInitStage(c *v2.MOSNConfig) {
	InitDefaultPath(c)
	InitDebugServe(c)
	InitializePidFile(c)
	InitializeTracing(c)
	InitializePlugin(c)
	InitializeWasm(c)
	InitializeThirdPartCodec(c)
}

// Default Pre-start Stage wrappers
func DefaultPreStartStage(mosn stagemanager.Application) {
	m := mosn.(*Mosn)

	// start xds client
	_ = m.StartXdsClient()
	featuregate.FinallyInitFunc()
	m.HandleExtendConfig()
}

// Default Start Stage wrappers
func DefaultStartStage(mosn stagemanager.Application) {
	m := mosn.(*Mosn)
	// register admin server
	// admin server should register after all prepares action ready
	srv := admin.Server{}
	srv.Start(m.Config)
}
