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

package rules

import (
	"errors"
	"sync"

	"mosn.io/mosn/pkg/log"
)

var tranferRuleInstance *TransferRuleManager

func init() {
	tranferRuleInstance = NewTransferRuleManager()
}

type TransferRuleManager struct {
	rules sync.Map
}

func NewTransferRuleManager() *TransferRuleManager {
	return &TransferRuleManager{}
}

func GetInstanceTransferRuleManger() *TransferRuleManager {
	return tranferRuleInstance
}

func (trm *TransferRuleManager) AddOrUpdateTransferRule(listenerName string, config []*TransferRuleConfig) error {
	if config == nil {
		return errors.New("rule is empty")
	}
	if rule, ok := trm.rules.LoadOrStore(listenerName, config); ok {
		if log.DefaultLogger.GetLogLevel() >= log.INFO {
			log.DefaultLogger.Infof("update transfer rule,old:%v,new:%s", rule, config)
		}
	}
	if log.DefaultLogger.GetLogLevel() >= log.INFO {
		log.DefaultLogger.Infof("add transfer rule,rule:%s", config)
	}
	return nil
}

func (trm *TransferRuleManager) DeleteTransferRule(listenerName string) {
	trm.rules.Delete(listenerName)
}

func (trm *TransferRuleManager) GetTransferRule(listenerName string) ([]*TransferRuleConfig, bool) {
	config, ok := trm.rules.Load(listenerName)
	if ok {
		return config.([]*TransferRuleConfig), true
	}
	return nil, false
}
