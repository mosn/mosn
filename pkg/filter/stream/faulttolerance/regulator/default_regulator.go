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

package regulator

import (
	"sync"

	v2 "mosn.io/mosn/pkg/config/v2"
)

type DefaultRegulator struct {
	measureModels *sync.Map
	workPool      WorkPool
	config        *v2.FaultToleranceFilterConfig
}

func NewDefaultRegulator(config *v2.FaultToleranceFilterConfig) *DefaultRegulator {
	regulator := &DefaultRegulator{
		measureModels: new(sync.Map),
		workPool:      NewDefaultWorkPool(config.TaskSize),
		config:        config,
	}
	return regulator
}

func (r *DefaultRegulator) Regulate(stat *InvocationStat) {
	if measureModel := r.createRegulationModel(stat); measureModel != nil {
		r.workPool.Schedule(measureModel)
	}
}

func (r *DefaultRegulator) createRegulationModel(stat *InvocationStat) *MeasureModel {
	key := stat.GetMeasureKey()
	if value, ok := r.measureModels.Load(key); ok {
		value.(*MeasureModel).AddInvocationStat(stat)
		return nil
	} else {
		measureModel := NewMeasureModel(key, r.config)
		measureModel.AddInvocationStat(stat)
		if value, ok := r.measureModels.LoadOrStore(key, measureModel); ok {
			value.(*MeasureModel).AddInvocationStat(stat)
			return nil
		} else {
			return value.(*MeasureModel)
		}
	}
}
