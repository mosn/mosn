// Copyright 1999-2020 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package system

import (
	"reflect"
	"sync"

	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
)

type RuleMap map[MetricType][]*Rule

// const
var (
	ruleMap       = make(RuleMap)
	ruleMapMux    = new(sync.RWMutex)
	currentRules  = make([]*Rule, 0)
	updateRuleMux = new(sync.Mutex)
)

// GetRules returns all the rules based on copy.
// It doesn't take effect for system module if user changes the rule.
// GetRules need to compete system module's global lock and the high performance losses of copy,
// 		reduce or do not call GetRules if possible
func GetRules() []Rule {
	rules := make([]*Rule, 0, len(ruleMap))
	ruleMapMux.RLock()
	for _, rs := range ruleMap {
		rules = append(rules, rs...)
	}
	ruleMapMux.RUnlock()

	ret := make([]Rule, 0, len(rules))
	for _, r := range rules {
		ret = append(ret, *r)
	}
	return ret
}

// getRules returns all the rules。Any changes of rules take effect for system module
// getRules is an internal interface.
func getRules() []*Rule {
	ruleMapMux.RLock()
	defer ruleMapMux.RUnlock()

	rules := make([]*Rule, 0, 8)
	for _, rs := range ruleMap {
		rules = append(rules, rs...)
	}
	return rules
}

// LoadRules loads given system rules to the rule manager, while all previous rules will be replaced.
func LoadRules(rules []*Rule) (bool, error) {
	updateRuleMux.Lock()
	defer updateRuleMux.Unlock()
	isEqual := reflect.DeepEqual(currentRules, rules)
	if isEqual {
		logging.Info("[System] Load rules is the same with current rules, so ignore load operation.")
		return false, nil
	}

	m := buildRuleMap(rules)

	if err := onRuleUpdate(m); err != nil {
		logging.Error(err, "Fail to load rules in system.LoadRules()", "rules", rules)
		return false, err
	}
	currentRules = rules
	return true, nil
}

// ClearRules clear all the previous rules
func ClearRules() error {
	_, err := LoadRules(nil)
	return err
}

func onRuleUpdate(r RuleMap) error {
	start := util.CurrentTimeNano()
	ruleMapMux.Lock()
	ruleMap = r
	ruleMapMux.Unlock()

	logging.Debug("[System onRuleUpdate] Time statistic(ns) for updating system rule", "timeCost", util.CurrentTimeNano()-start)
	if len(r) > 0 {
		logging.Info("[SystemRuleManager] System rules loaded", "rules", r)
	} else {
		logging.Info("[SystemRuleManager] System rules were cleared")
	}
	return nil
}

func buildRuleMap(rules []*Rule) RuleMap {
	m := make(RuleMap)

	if len(rules) == 0 {
		return m
	}

	for _, rule := range rules {
		if err := IsValidSystemRule(rule); err != nil {
			logging.Warn("[System buildRuleMap] Ignoring invalid system rule", "rule", rule, "err", err.Error())
			continue
		}
		rulesOfRes, exists := m[rule.MetricType]
		if !exists {
			m[rule.MetricType] = []*Rule{rule}
		} else {
			m[rule.MetricType] = append(rulesOfRes, rule)
		}
	}
	return m
}

// IsValidSystemRule determine the system rule is valid or not
func IsValidSystemRule(rule *Rule) error {
	if rule == nil {
		return errors.New("nil Rule")
	}
	if rule.TriggerCount < 0 {
		return errors.New("negative threshold")
	}
	if rule.MetricType >= MetricTypeSize {
		return errors.New("invalid metric type")
	}

	if rule.MetricType == CpuUsage && rule.TriggerCount > 1 {
		return errors.New("invalid CPU usage, valid range is [0.0, 1.0]")
	}
	return nil
}
