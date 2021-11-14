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

//
// list
// 	1. create match
//  2. match
//  3. false->loop
// 	4. true route
