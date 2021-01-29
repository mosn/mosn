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

package isolation

import (
	"reflect"
	"sync"

	"github.com/alibaba/sentinel-golang/core/misc"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
)

var (
	ruleMap       = make(map[string][]*Rule)
	rwMux         = &sync.RWMutex{}
	currentRules  = make([]*Rule, 0)
	updateRuleMux = new(sync.Mutex)
)

// LoadRules loads the given isolation rules to the rule manager, while all previous rules will be replaced.
// the first returned value indicates whether do real load operation, if the rules is the same with previous rules, return false
func LoadRules(rules []*Rule) (bool, error) {
	updateRuleMux.Lock()
	defer updateRuleMux.Unlock()
	isEqual := reflect.DeepEqual(currentRules, rules)
	if isEqual {
		logging.Info("[Isolation] Load rules is the same with current rules, so ignore load operation.")
		return false, nil
	}

	err := onRuleUpdate(rules)
	return true, err
}

func onRuleUpdate(rules []*Rule) (err error) {
	m := make(map[string][]*Rule, len(rules))
	for _, r := range rules {
		if e := IsValid(r); e != nil {
			logging.Error(e, "Invalid isolation rule in isolation.LoadRules()", "rule", r)
			continue
		}
		resRules, ok := m[r.Resource]
		if !ok {
			resRules = make([]*Rule, 0, 1)
		}
		m[r.Resource] = append(resRules, r)
	}

	start := util.CurrentTimeNano()

	for res, rs := range m {
		if len(rs) > 0 {
			// update resource slot chain
			misc.RegisterRuleCheckSlotForResource(res, DefaultSlot)
		}
	}
	rwMux.Lock()
	ruleMap = m
	currentRules = rules
	rwMux.Unlock()

	logging.Debug("[Isolation LoadRules] Time statistic(ns) for updating isolation rule", "timeCost", util.CurrentTimeNano()-start)
	logRuleUpdate(m)
	return
}

// ClearRules clears all the rules in isolation module.
func ClearRules() error {
	_, err := LoadRules(nil)
	return err
}

// GetRules returns all the rules based on copy.
// It doesn't take effect for isolation module if user changes the rule.
func GetRules() []Rule {
	rules := getRules()
	ret := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, *rule)
	}
	return ret
}

// GetRulesOfResource returns specific resource's rules based on copy.
// It doesn't take effect for isolation module if user changes the rule.
func GetRulesOfResource(res string) []Rule {
	rules := getRulesOfResource(res)
	ret := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, *rule)
	}
	return ret
}

// getRules returns all the rules。Any changes of rules take effect for isolation module
// getRules is an internal interface.
func getRules() []*Rule {
	rwMux.RLock()
	defer rwMux.RUnlock()

	return rulesFrom(ruleMap)
}

// getRulesOfResource returns specific resource's rules。Any changes of rules take effect for isolation module
// getRulesOfResource is an internal interface.
func getRulesOfResource(res string) []*Rule {
	rwMux.RLock()
	defer rwMux.RUnlock()

	resRules, exist := ruleMap[res]
	if !exist {
		return nil
	}
	ret := make([]*Rule, 0, len(resRules))
	for _, r := range resRules {
		ret = append(ret, r)
	}
	return ret
}

func rulesFrom(m map[string][]*Rule) []*Rule {
	rules := make([]*Rule, 0, 8)
	if len(m) == 0 {
		return rules
	}
	for _, rs := range m {
		for _, r := range rs {
			if r != nil {
				rules = append(rules, r)
			}
		}
	}
	return rules
}

func logRuleUpdate(m map[string][]*Rule) {
	rs := rulesFrom(m)
	if len(rs) == 0 {
		logging.Info("[IsolationRuleManager] Isolation rules were cleared")
	} else {
		logging.Info("[IsolationRuleManager] Isolation rules were loaded", "rules", rs)
	}
}

// IsValidRule checks whether the given Rule is valid.
func IsValid(r *Rule) error {
	if r == nil {
		return errors.New("nil isolation rule")
	}
	if len(r.Resource) == 0 {
		return errors.New("empty resource of isolation rule")
	}
	if r.MetricType != Concurrency {
		return errors.Errorf("unsupported metric type: %d", r.MetricType)
	}
	if r.Threshold == 0 {
		return errors.New("zero threshold")
	}
	return nil
}
