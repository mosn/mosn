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

package hotspot

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/alibaba/sentinel-golang/core/misc"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
)

// TrafficControllerGenFunc represents the TrafficShapingController generator function of a specific control behavior.
type TrafficControllerGenFunc func(r *Rule, reuseMetric *ParamsMetric) TrafficShapingController

// trafficControllerMap represents the map storage for TrafficShapingController.
type trafficControllerMap map[string][]TrafficShapingController

var (
	tcGenFuncMap  = make(map[ControlBehavior]TrafficControllerGenFunc, 4)
	tcMap         = make(trafficControllerMap)
	tcMux         = new(sync.RWMutex)
	currentRules  = make([]*Rule, 0)
	updateRuleMux = new(sync.Mutex)
)

func init() {
	// Initialize the traffic shaping controller generator map for existing control behaviors.
	tcGenFuncMap[Reject] = func(r *Rule, reuseMetric *ParamsMetric) TrafficShapingController {
		var baseTc *baseTrafficShapingController
		if reuseMetric != nil {
			// new BaseTrafficShapingController with reuse statistic metric
			baseTc = newBaseTrafficShapingControllerWithMetric(r, reuseMetric)
		} else {
			baseTc = newBaseTrafficShapingController(r)
		}
		if baseTc == nil {
			return nil
		}
		return &rejectTrafficShapingController{
			baseTrafficShapingController: *baseTc,
			burstCount:                   r.BurstCount,
		}
	}

	tcGenFuncMap[Throttling] = func(r *Rule, reuseMetric *ParamsMetric) TrafficShapingController {
		var baseTc *baseTrafficShapingController
		if reuseMetric != nil {
			baseTc = newBaseTrafficShapingControllerWithMetric(r, reuseMetric)
		} else {
			baseTc = newBaseTrafficShapingController(r)
		}
		if baseTc == nil {
			return nil
		}
		return &throttlingTrafficShapingController{
			baseTrafficShapingController: *baseTc,
			maxQueueingTimeMs:            r.MaxQueueingTimeMs,
		}
	}
}

func getTrafficControllersFor(res string) []TrafficShapingController {
	tcMux.RLock()
	defer tcMux.RUnlock()

	return tcMap[res]
}

// LoadRules replaces old rules with the given hotspot parameter flow control rules. Return value:
//
// bool: indicates whether the internal map has been changed;
// error: indicates whether occurs the error.
func LoadRules(rules []*Rule) (bool, error) {
	updateRuleMux.Lock()
	defer updateRuleMux.Unlock()
	isEqual := reflect.DeepEqual(currentRules, rules)
	if isEqual {
		logging.Info("[HotSpot] Load rules is the same with current rules, so ignore load operation.")
		return false, nil
	}

	err := onRuleUpdate(rules)
	return true, err
}

// GetRules returns all the rules based on copy.
// It doesn't take effect for hotspot module if user changes the rule.
// GetRules need to compete hotspot module's global lock and the high performance losses of copy,
// 		reduce or do not call GetRules if possible
func GetRules() []Rule {
	tcMux.RLock()
	rules := rulesFrom(tcMap)
	tcMux.RUnlock()

	ret := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, *rule)
	}
	return ret
}

// GetRulesOfResource returns specific resource's rules based on copy.
// It doesn't take effect for hotspot module if user changes the rule.
// GetRulesOfResource need to compete hotspot module's global lock and the high performance losses of copy,
// 		reduce or do not call GetRulesOfResource frequently if possible
func GetRulesOfResource(res string) []Rule {
	tcMux.RLock()
	resTcs := tcMap[res]
	tcMux.RUnlock()

	ret := make([]Rule, 0, len(resTcs))
	for _, tc := range resTcs {
		ret = append(ret, *tc.BoundRule())
	}
	return ret
}

// ClearRules clears all parameter flow rules.
func ClearRules() error {
	_, err := LoadRules(nil)
	return err
}

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

	newRuleMap := make(map[string][]*Rule, len(rules))
	for _, r := range rules {
		if err := IsValidRule(r); err != nil {
			logging.Warn("[HotSpot onRuleUpdate] Ignoring invalid hotspot rule when loading new rules", "rule", r, "err", err.Error())
			continue
		}
		if r.Mode == "" || r.Mode == CLOSE {
			logging.Warn("[HotSpot onRuleUpdate] Ignoring Close hotspot rule when loading new rules", "rule", r)
			continue
		}
		res := r.ResourceName()
		ruleSet, ok := newRuleMap[res]
		if !ok {
			ruleSet = make([]*Rule, 0, 1)
		}
		ruleSet = append(ruleSet, r)
		newRuleMap[res] = ruleSet
	}

	m := make(trafficControllerMap)
	for res, rules := range newRuleMap {
		m[res] = make([]TrafficShapingController, 0, len(rules))
	}

	start := util.CurrentTimeNano()

	tcMux.RLock()
	tcMapClone := make(trafficControllerMap, len(tcMap))
	for res, tcs := range tcMap {
		resTcClone := make([]TrafficShapingController, 0, len(tcs))
		resTcClone = append(resTcClone, tcs...)
		tcMapClone[res] = resTcClone
	}
	tcMux.RUnlock()

	for res, resRules := range newRuleMap {
		emptyTcList := make([]TrafficShapingController, 0, 0)
		for _, r := range resRules {
			oldResTcs := tcMapClone[res]
			if oldResTcs == nil {
				oldResTcs = emptyTcList
			}

			equalIdx, reuseStatIdx := calculateReuseIndexFor(r, oldResTcs)
			// there is equivalent rule in old traffic shaping controller slice
			if equalIdx >= 0 {
				equalOldTC := oldResTcs[equalIdx]
				insertTcToTcMap(equalOldTC, res, m)
				// remove old tc from old resTcs
				tcMapClone[res] = append(oldResTcs[:equalIdx], oldResTcs[equalIdx+1:]...)
				continue
			}

			// generate new traffic shaping controller
			generator, supported := tcGenFuncMap[r.ControlBehavior]
			if !supported {
				logging.Warn("[HotSpot onRuleUpdate] Ignoring the frequent param flow rule due to unsupported control behavior", "rule", r)
				continue
			}
			var tc TrafficShapingController
			if reuseStatIdx >= 0 {
				// generate new traffic shaping controller with reusable statistic metric.
				tc = generator(r, oldResTcs[reuseStatIdx].BoundMetric())
			} else {
				tc = generator(r, nil)
			}
			if tc == nil {
				logging.Debug("[HotSpot onRuleUpdate] Ignoring the frequent param flow rule due to bad generated traffic controller", "rule", r)
				continue
			}

			//  remove the reused traffic shaping controller old res tcs
			if reuseStatIdx >= 0 {
				tcMapClone[res] = append(oldResTcs[:reuseStatIdx], oldResTcs[reuseStatIdx+1:]...)
			}
			insertTcToTcMap(tc, res, m)
		}
	}
	for res, tcs := range m {
		if len(tcs) > 0 {
			// update resource slot chain
			misc.RegisterRuleCheckSlotForResource(res, DefaultSlot)
			for _, tc := range tcs {
				if tc.BoundRule().MetricType == Concurrency {
					misc.RegisterStatSlotForResource(res, DefaultConcurrencyStatSlot)
					break
				}
			}
		}
	}

	tcMux.Lock()
	tcMap = m
	currentRules = rules
	tcMux.Unlock()

	logging.Debug("[HotSpot onRuleUpdate] Time statistic(ns) for updating hotSpot rule", "timeCost", util.CurrentTimeNano()-start)
	logRuleUpdate(newRuleMap)
	return nil
}

func logRuleUpdate(m map[string][]*Rule) {
	rules := make([]*Rule, 0, 8)
	for _, rs := range m {
		if len(rs) == 0 {
			continue
		}
		rules = append(rules, rs...)
	}
	if len(rules) == 0 {
		logging.Info("[HotspotRuleManager] Hotspot rules were cleared")
	} else {
		logging.Info("[HotspotRuleManager] Hotspot rules were loaded", "rules", rules)
	}
}

func rulesFrom(m trafficControllerMap) []*Rule {
	rules := make([]*Rule, 0, 8)
	if len(m) == 0 {
		return rules
	}
	for _, rs := range m {
		if len(rs) == 0 {
			continue
		}
		for _, r := range rs {
			if r != nil && r.BoundRule() != nil {
				rules = append(rules, r.BoundRule())
			}
		}
	}
	return rules
}

func calculateReuseIndexFor(r *Rule, oldResTcs []TrafficShapingController) (equalIdx, reuseStatIdx int) {
	// the index of equivalent rule in old traffic shaping controller slice
	equalIdx = -1
	// the index of statistic reusable rule in old traffic shaping controller slice
	reuseStatIdx = -1

	for idx, oldTc := range oldResTcs {
		oldRule := oldTc.BoundRule()
		if oldRule.Equals(r) {
			// break if there is equivalent rule
			equalIdx = idx
			break
		}
		// find the index of first StatReusable rule
		if !oldRule.IsStatReusable(r) {
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

func insertTcToTcMap(tc TrafficShapingController, res string, m trafficControllerMap) {
	tcsOfRes, exists := m[res]
	if !exists {
		tcsOfRes = make([]TrafficShapingController, 0, 1)
		m[res] = append(tcsOfRes, tc)
	} else {
		m[res] = append(tcsOfRes, tc)
	}
}

func IsValidRule(rule *Rule) error {
	if rule == nil {
		return errors.New("nil hotspot Rule")
	}
	if len(rule.Resource) == 0 {
		return errors.New("empty resource name")
	}
	if rule.Threshold < 0 {
		return errors.New("negative threshold")
	}
	if rule.MetricType < 0 {
		return errors.New("invalid metric type")
	}
	if rule.ControlBehavior < 0 {
		return errors.New("invalid control strategy")
	}
	if rule.ParamIndex < 0 {
		return errors.New("invalid param index")
	}
	if rule.MetricType == QPS && rule.DurationInSec <= 0 {
		return errors.New("invalid duration")
	}
	return checkControlBehaviorField(rule)
}

func checkControlBehaviorField(rule *Rule) error {
	switch rule.ControlBehavior {
	case Reject:
		if rule.BurstCount < 0 {
			return errors.New("invalid BurstCount")
		}
		return nil
	case Throttling:
		if rule.MaxQueueingTimeMs < 0 {
			return errors.New("invalid MaxQueueingTimeMs")
		}
		return nil
	default:
	}
	return nil
}

// SetTrafficShapingGenerator sets the traffic controller generator for the given control behavior.
// Note that modifying the generator of default control behaviors is not allowed.
func SetTrafficShapingGenerator(cb ControlBehavior, generator TrafficControllerGenFunc) error {
	if generator == nil {
		return errors.New("nil generator")
	}
	if cb >= Reject && cb <= Throttling {
		return errors.New("not allowed to replace the generator for default control behaviors")
	}
	tcMux.Lock()
	defer tcMux.Unlock()

	tcGenFuncMap[cb] = generator
	return nil
}

func RemoveTrafficShapingGenerator(cb ControlBehavior) error {
	if cb >= Reject && cb <= Throttling {
		return errors.New("not allowed to replace the generator for default control behaviors")
	}
	tcMux.Lock()
	defer tcMux.Unlock()

	delete(tcGenFuncMap, cb)
	return nil
}
