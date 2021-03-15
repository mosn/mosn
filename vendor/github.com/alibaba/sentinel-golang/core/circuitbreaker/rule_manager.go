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

package circuitbreaker

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/alibaba/sentinel-golang/core/misc"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
)

type CircuitBreakerGenFunc func(r *Rule, reuseStat interface{}) (CircuitBreaker, error)

var (
	cbGenFuncMap = make(map[Strategy]CircuitBreakerGenFunc, 4)

	breakerRules  = make(map[string][]*Rule)
	breakers      = make(map[string][]CircuitBreaker)
	updateMux     = &sync.RWMutex{}
	currentRules  = make([]*Rule, 0)
	updateRuleMux = new(sync.Mutex)

	stateChangeListeners = make([]StateChangeListener, 0)
)

func init() {
	cbGenFuncMap[SlowRequestRatio] = func(r *Rule, reuseStat interface{}) (CircuitBreaker, error) {
		if r == nil {
			return nil, errors.New("nil rule")
		}
		if reuseStat == nil {
			return newSlowRtCircuitBreaker(r)
		}
		stat, ok := reuseStat.(*slowRequestLeapArray)
		if !ok || stat == nil {
			logging.Warn("[CircuitBreaker RuleManager] Expect to generate circuit breaker with reuse statistic, but fail to do type assertion, expect:*slowRequestLeapArray", "statType", reflect.TypeOf(stat).Name())
			return newSlowRtCircuitBreaker(r)
		}
		return newSlowRtCircuitBreakerWithStat(r, stat), nil
	}

	cbGenFuncMap[ErrorRatio] = func(r *Rule, reuseStat interface{}) (CircuitBreaker, error) {
		if r == nil {
			return nil, errors.New("nil rule")
		}
		if reuseStat == nil {
			return newErrorRatioCircuitBreaker(r)
		}
		stat, ok := reuseStat.(*errorCounterLeapArray)
		if !ok || stat == nil {
			logging.Warn("[CircuitBreaker RuleManager] Expect to generate circuit breaker with reuse statistic, but fail to do type assertion, expect:*errorCounterLeapArray", "statType", reflect.TypeOf(stat).Name())
			return newErrorRatioCircuitBreaker(r)
		}
		return newErrorRatioCircuitBreakerWithStat(r, stat), nil
	}

	cbGenFuncMap[ErrorCount] = func(r *Rule, reuseStat interface{}) (CircuitBreaker, error) {
		if r == nil {
			return nil, errors.New("nil rule")
		}
		if reuseStat == nil {
			return newErrorCountCircuitBreaker(r)
		}
		stat, ok := reuseStat.(*errorCounterLeapArray)
		if !ok || stat == nil {
			logging.Warn("[CircuitBreaker RuleManager] Expect to generate circuit breaker with reuse statistic, but fail to do type assertion, expect:*errorCounterLeapArray", "statType", reflect.TypeOf(stat).Name())
			return newErrorCountCircuitBreaker(r)
		}
		return newErrorCountCircuitBreakerWithStat(r, stat), nil
	}
}

// GetRulesOfResource returns specific resource's rules based on copy.
// It doesn't take effect for circuit breaker module if user changes the rule.
// GetRulesOfResource need to compete circuit breaker module's global lock and the high performance losses of copy,
// 		reduce or do not call GetRulesOfResource frequently if possible
func GetRulesOfResource(resource string) []Rule {
	updateMux.RLock()
	resRules, ok := breakerRules[resource]
	updateMux.RUnlock()
	if !ok {
		return nil
	}
	ret := make([]Rule, 0, len(resRules))
	for _, rule := range resRules {
		ret = append(ret, *rule)
	}
	return ret
}

// GetRules returns all the rules based on copy.
// It doesn't take effect for circuit breaker module if user changes the rule.
// GetRules need to compete circuit breaker module's global lock and the high performance losses of copy,
// 		reduce or do not call GetRules if possible
func GetRules() []Rule {
	updateMux.RLock()
	rules := rulesFrom(breakerRules)
	updateMux.RUnlock()
	ret := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, *rule)
	}
	return ret
}

// ClearRules clear all the previous rules.
func ClearRules() error {
	_, err := LoadRules(nil)
	return err
}

// LoadRules replaces old rules with the given circuit breaking rules.
//
// return value:
//
// bool: was designed to indicate whether the internal map has been changed
// error: was designed to indicate whether occurs the error.
func LoadRules(rules []*Rule) (bool, error) {
	// TODO in order to avoid invalid update, should check consistent with last update rules
	updateRuleMux.Lock()
	defer updateRuleMux.Unlock()
	isEqual := reflect.DeepEqual(currentRules, rules)
	if isEqual {
		logging.Info("[CircuitBreaker] Load rules is the same with current rules, so ignore load operation.")
		return false, nil
	}

	err := onRuleUpdate(rules)
	return true, err
}

func getBreakersOfResource(resource string) []CircuitBreaker {
	updateMux.RLock()
	resCBs := breakers[resource]
	updateMux.RUnlock()
	ret := make([]CircuitBreaker, 0, len(resCBs))
	if len(resCBs) == 0 {
		return ret
	}
	ret = append(ret, resCBs...)
	return ret
}

func calculateReuseIndexFor(r *Rule, oldResCbs []CircuitBreaker) (equalIdx, reuseStatIdx int) {
	// the index of equivalent rule in old circuit breaker slice
	equalIdx = -1
	// the index of statistic reusable rule in old circuit breaker slice
	reuseStatIdx = -1

	for idx, oldTc := range oldResCbs {
		oldRule := oldTc.BoundRule()
		if oldRule.isEqualsTo(r) {
			// break if there is equivalent rule
			equalIdx = idx
			break
		}
		// find the index of first StatReusable rule
		if !oldRule.isStatReusable(r) {
			continue
		}
		if reuseStatIdx >= 0 {
			// had find reuse rule.
			continue
		}
		reuseStatIdx = idx
	}
	return equalIdx, reuseStatIdx
}

func insertCbToCbMap(cb CircuitBreaker, res string, m map[string][]CircuitBreaker) {
	cbsOfRes, exists := m[res]
	if !exists {
		cbsOfRes = make([]CircuitBreaker, 0, 1)
		m[res] = append(cbsOfRes, cb)
	} else {
		m[res] = append(cbsOfRes, cb)
	}
}

// Concurrent safe to update rules
func onRuleUpdate(rules []*Rule) (err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("%+v", r)
			}
		}
	}()

	newBreakerRules := make(map[string][]*Rule, len(rules))
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if err := IsValid(rule); err != nil {
			logging.Warn("[CircuitBreaker onRuleUpdate] Ignoring invalid circuit breaking rule when loading new rules", "rule", rule, "err", err.Error())
			continue
		}

		classification := rule.ResourceName()
		ruleSet, ok := newBreakerRules[classification]
		if !ok {
			ruleSet = make([]*Rule, 0, 1)
		}
		ruleSet = append(ruleSet, rule)
		newBreakerRules[classification] = ruleSet
	}

	newBreakers := make(map[string][]CircuitBreaker, len(rules))
	// in order to avoid growing, build newBreakers in advance
	for res, rules := range newBreakerRules {
		newBreakers[res] = make([]CircuitBreaker, 0, len(rules))
	}

	start := util.CurrentTimeNano()

	updateMux.RLock()
	breakersClone := make(map[string][]CircuitBreaker, len(newBreakerRules))
	for res, tcs := range breakers {
		resTcClone := make([]CircuitBreaker, 0, len(tcs))
		resTcClone = append(resTcClone, tcs...)
		breakersClone[res] = resTcClone
	}
	updateMux.RUnlock()

	for res, resRules := range newBreakerRules {
		emptyCircuitBreakerList := make([]CircuitBreaker, 0, 0)
		for _, r := range resRules {
			oldResCbs := breakersClone[res]
			if oldResCbs == nil {
				oldResCbs = emptyCircuitBreakerList
			}
			equalIdx, reuseStatIdx := calculateReuseIndexFor(r, oldResCbs)

			// First check equals scenario
			if equalIdx >= 0 {
				// reuse the old cb
				equalOldCb := oldResCbs[equalIdx]
				insertCbToCbMap(equalOldCb, res, newBreakers)
				// remove old cb from oldResCbs
				breakersClone[res] = append(oldResCbs[:equalIdx], oldResCbs[equalIdx+1:]...)
				continue
			}

			generator := cbGenFuncMap[r.Strategy]
			if generator == nil {
				logging.Warn("[CircuitBreaker onRuleUpdate] Ignoring the rule due to unsupported circuit breaking strategy", "rule", r)
				continue
			}

			var cb CircuitBreaker
			var e error
			if reuseStatIdx >= 0 {
				cb, e = generator(r, oldResCbs[reuseStatIdx].BoundStat())
			} else {
				cb, e = generator(r, nil)
			}
			if cb == nil || e != nil {
				logging.Warn("[CircuitBreaker onRuleUpdate] Ignoring the rule due to bad generated circuit breaker", "rule", r, "err", e.Error())
				continue
			}

			if reuseStatIdx >= 0 {
				breakersClone[res] = append(oldResCbs[:reuseStatIdx], oldResCbs[reuseStatIdx+1:]...)
			}
			insertCbToCbMap(cb, res, newBreakers)
		}
	}

	for res, cbs := range newBreakers {
		if len(cbs) > 0 {
			// update resource slot chain
			misc.RegisterRuleCheckSlotForResource(res, DefaultSlot)
			misc.RegisterStatSlotForResource(res, DefaultMetricStatSlot)
		}
	}

	updateMux.Lock()
	breakerRules = newBreakerRules
	breakers = newBreakers
	currentRules = rules
	updateMux.Unlock()

	logging.Debug("[CircuitBreaker onRuleUpdate] Time statistics(ns) for updating circuit breaker rule", "timeCost", util.CurrentTimeNano()-start)
	logRuleUpdate(newBreakerRules)
	return nil
}

func rulesFrom(rm map[string][]*Rule) []*Rule {
	rules := make([]*Rule, 0, 8)
	if len(rm) == 0 {
		return rules
	}
	for _, rs := range rm {
		if len(rs) == 0 {
			continue
		}
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
		logging.Info("[CircuitBreakerRuleManager] Circuit breaking rules were cleared")
	} else {
		logging.Info("[CircuitBreakerRuleManager] Circuit breaking rules were loaded", "rules", rs)
	}
}

// RegisterStateChangeListeners registers the global state change listener for all circuit breakers
// Note: this function is not thread-safe.
func RegisterStateChangeListeners(listeners ...StateChangeListener) {
	if len(listeners) == 0 {
		return
	}

	stateChangeListeners = append(stateChangeListeners, listeners...)
}

// ClearStateChangeListeners clears the all StateChangeListener
// Note: this function is not thread-safe.
func ClearStateChangeListeners() {
	stateChangeListeners = make([]StateChangeListener, 0)
}

// SetCircuitBreakerGenerator sets the circuit breaker generator for the given strategy.
// Note that modifying the generator of default strategies is not allowed.
func SetCircuitBreakerGenerator(s Strategy, generator CircuitBreakerGenFunc) error {
	if generator == nil {
		return errors.New("nil generator")
	}
	if s <= ErrorCount {
		return errors.New("not allowed to replace the generator for default circuit breaking strategies")
	}
	updateMux.Lock()
	defer updateMux.Unlock()

	cbGenFuncMap[s] = generator
	return nil
}

func RemoveCircuitBreakerGenerator(s Strategy) error {
	if s <= ErrorCount {
		return errors.New("not allowed to remove the generator for default circuit breaking strategies")
	}
	updateMux.Lock()
	defer updateMux.Unlock()

	delete(cbGenFuncMap, s)
	return nil
}

func IsValid(r *Rule) error {
	if len(r.Resource) == 0 {
		return errors.New("empty resource name")
	}
	if r.StatIntervalMs <= 0 {
		return errors.New("invalid StatIntervalMs")
	}
	if r.RetryTimeoutMs <= 0 {
		return errors.New("invalid RetryTimeoutMs")
	}
	if r.Threshold < 0.0 {
		return errors.New("invalid Threshold")
	}
	if r.Strategy == SlowRequestRatio && r.Threshold > 1.0 {
		return errors.New("invalid slow request ratio threshold (valid range: [0.0, 1.0])")
	}
	if r.Strategy == ErrorRatio && r.Threshold > 1.0 {
		return errors.New("invalid error ratio threshold (valid range: [0.0, 1.0])")
	}
	if r.StatSlidingWindowBucketCount != 0 && r.StatIntervalMs%r.StatSlidingWindowBucketCount != 0 {
		logging.Warn("[CircuitBreaker IsValid] The following must be true: StatIntervalMs % StatSlidingWindowBucketCount == 0. StatSlidingWindowBucketCount will be replaced by 1", "rule", r)
	}
	return nil
}
