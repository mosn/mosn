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

package configmanager

import (
	"mosn.io/mosn/pkg/featuregate"
	"mosn.io/mosn/pkg/log"
)

const ConfigAutoWrite featuregate.Feature = "auto_config"

var enableAutoWrite bool = false
var feature *ConfigAutoFeature

// ConfigAutoFeature controls xDS update config will overwrite config file or not, default is not
type ConfigAutoFeature struct {
	featuregate.BaseFeatureSpec
}

func (f *ConfigAutoFeature) InitFunc() {
	log.DefaultLogger.Infof("auto write config when updated")
	enableAutoWrite = true
}

func init() {
	feature = &ConfigAutoFeature{
		BaseFeatureSpec: featuregate.BaseFeatureSpec{
			DefaultValue: false,
		},
	}
	featuregate.AddFeatureSpec(ConfigAutoWrite, feature)
}

func tryDump() {
	if enableAutoWrite {
		setDump()
	}
}
