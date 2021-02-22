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

package flow

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/config"
	"github.com/alibaba/sentinel-golang/core/misc"
	"github.com/alibaba/sentinel-golang/core/stat"
	sbase "github.com/alibaba/sentinel-golang/core/stat/base"
	"github.com/alibaba/sentinel-golang/logging"
	"github.com/alibaba/sentinel-golang/util"
	"github.com/pkg/errors"
)

// TrafficControllerGenFunc represents the TrafficShapingController generator function of a specific control behavior.
type TrafficControllerGenFunc func(*Rule, *standaloneStatistic) (*TrafficShapingController, error)

type trafficControllerGenKey struct {
	tokenCalculateStrategy TokenCalculateStrategy
	controlBehavior        ControlBehavior
}

// TrafficControllerMap represents the map storage for TrafficShapingController.
type TrafficControllerMap map[string][]*TrafficShapingController

var (
	tcGenFuncMap = make(map[trafficControllerGenKey]TrafficControllerGenFunc, 4)
	tcMap        = make(TrafficControllerMap)
	tcMux        = new(sync.RWMutex)
	nopStat      = &standaloneStatistic{
		reuseResourceStat: false,
		readOnlyMetric:    base.NopReadStat(),
		writeOnlyMetric:   base.NopWriteStat(),
	}
	currentRules  = make([]*Rule, 0)
	updateRuleMux = new(sync.Mutex)
)

func init() {
	// Initialize the traffic shaping controller generator map for existing control behaviors.
	tcGenFuncMap[trafficControllerGenKey{
		tokenCalculateStrategy: Direct,
		controlBehavior:        Reject,
	}] = func(rule *Rule, boundStat *standaloneStatistic) (*TrafficShapingController, error) {
		if boundStat == nil {
			var err error
			boundStat, err = generateStatFor(rule)
			if err != nil {
				return nil, err
			}
		}
		tsc, err := NewTrafficShapingController(rule, boundStat)
		if err != nil || tsc == nil {
			return nil, err
		}
		tsc.flowCalculator = NewDirectTrafficShapingCalculator(tsc, rule.Threshold)
		tsc.flowChecker = NewRejectTrafficShapingChecker(tsc, rule)
		return tsc, nil
	}
	tcGenFuncMap[trafficControllerGenKey{
		tokenCalculateStrategy: Direct,
		controlBehavior:        Throttling,
	}] = func(rule *Rule, _ *standaloneStatistic) (*TrafficShapingController, error) {
		// Direct token calculate strategy and throttling control behavior don't use stat, so we just give a nop stat.
		tsc, err := NewTrafficShapingController(rule, nopStat)
		if err != nil || tsc == nil {
			return nil, err
		}
		tsc.flowCalculator = NewDirectTrafficShapingCalculator(tsc, rule.Threshold)
		tsc.flowChecker = NewThrottlingChecker(tsc, rule.MaxQueueingTimeMs, rule.StatIntervalInMs)
		return tsc, nil
	}
	tcGenFuncMap[trafficControllerGenKey{
		tokenCalculateStrategy: WarmUp,
		controlBehavior:        Reject,
	}] = func(rule *Rule, boundStat *standaloneStatistic) (*TrafficShapingController, error) {
		if boundStat == nil {
			var err error
			boundStat, err = generateStatFor(rule)
			if err != nil {
				return nil, err
			}
		}
		tsc, err := NewTrafficShapingController(rule, boundStat)
		if err != nil || tsc == nil {
			return nil, err
		}
		tsc.flowCalculator = NewWarmUpTrafficShapingCalculator(tsc, rule)
		tsc.flowChecker = NewRejectTrafficShapingChecker(tsc, rule)
		return tsc, nil
	}
	tcGenFuncMap[trafficControllerGenKey{
		tokenCalculateStrategy: WarmUp,
		controlBehavior:        Throttling,
	}] = func(rule *Rule, boundStat *standaloneStatistic) (*TrafficShapingController, error) {
		if boundStat == nil {
			var err error
			boundStat, err = generateStatFor(rule)
			if err != nil {
				return nil, err
			}
		}
		tsc, err := NewTrafficShapingController(rule, boundStat)
		if err != nil || tsc == nil {
			return nil, err
		}
		tsc.flowCalculator = NewWarmUpTrafficShapingCalculator(tsc, rule)
		tsc.flowChecker = NewThrottlingChecker(tsc, rule.MaxQueueingTimeMs, rule.StatIntervalInMs)
		return tsc, nil
	}
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
		logging.Info("[FlowRuleManager] Flow rules were cleared")
	} else {
		logging.Info("[FlowRuleManager] Flow rules were loaded", "rules", rules)
	}
}

func onRuleUpdate(rules []*Rule) (err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}
		}
	}()

	resRulesMap := make(map[string][]*Rule, len(rules))
	for _, rule := range rules {
		if err := IsValidRule(rule); err != nil {
			logging.Warn("[Flow onRuleUpdate] Ignoring invalid flow rule", "rule", rule, "reason", err.Error())
			continue
		}
		resRules, exist := resRulesMap[rule.Resource]
		if !exist {
			resRules = make([]*Rule, 0, 1)
		}
		resRulesMap[rule.Resource] = append(resRules, rule)
	}
	m := make(TrafficControllerMap, len(resRulesMap))
	start := util.CurrentTimeNano()

	tcMux.RLock()
	tcMapClone := make(TrafficControllerMap, len(resRulesMap))
	for res, tcs := range tcMap {
		resTcClone := make([]*TrafficShapingController, 0, len(tcs))
		resTcClone = append(resTcClone, tcs...)
		tcMapClone[res] = resTcClone
	}
	tcMux.RUnlock()

	for res, rulesOfRes := range resRulesMap {
		m[res] = buildRulesOfRes(res, rulesOfRes, tcMapClone)
	}

	for res, tcs := range m {
		if len(tcs) > 0 {
			// update resource slot chain
			misc.RegisterRuleCheckSlotForResource(res, DefaultSlot)
			misc.RegisterStatSlotForResource(res, DefaultStandaloneStatSlot)
		}
	}

	tcMux.Lock()
	tcMap = m
	currentRules = rules
	tcMux.Unlock()

	logging.Debug("[Flow onRuleUpdate] Time statistic(ns) for updating flow rule", "timeCost", util.CurrentTimeNano()-start)
	logRuleUpdate(resRulesMap)
	return nil
}

// LoadRules loads the given flow rules to the rule manager, while all previous rules will be replaced.
// the first returned value indicates whether do real load operation, if the rules is the same with previous rules, return false
func LoadRules(rules []*Rule) (bool, error) {
	// TODO: rethink the design
	updateRuleMux.Lock()
	defer updateRuleMux.Unlock()
	isEqual := reflect.DeepEqual(currentRules, rules)
	if isEqual {
		logging.Info("[Flow] Load rules is the same with current rules, so ignore load operation.")
		return false, nil
	}

	err := onRuleUpdate(rules)
	return true, err
}

// getRules returns all the rules。Any changes of rules take effect for flow module
// getRules is an internal interface.
func getRules() []*Rule {
	tcMux.RLock()
	defer tcMux.RUnlock()

	return rulesFrom(tcMap)
}

// getRulesOfResource returns specific resource's rules。Any changes of rules take effect for flow module
// getRulesOfResource is an internal interface.
func getRulesOfResource(res string) []*Rule {
	tcMux.RLock()
	defer tcMux.RUnlock()

	resTcs, exist := tcMap[res]
	if !exist {
		return nil
	}
	ret := make([]*Rule, 0, len(resTcs))
	for _, tc := range resTcs {
		ret = append(ret, tc.BoundRule())
	}
	return ret
}

// GetRules returns all the rules based on copy.
// It doesn't take effect for flow module if user changes the rule.
func GetRules() []Rule {
	rules := getRules()
	ret := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, *rule)
	}
	return ret
}

// GetRulesOfResource returns specific resource's rules based on copy.
// It doesn't take effect for flow module if user changes the rule.
func GetRulesOfResource(res string) []Rule {
	rules := getRulesOfResource(res)
	ret := make([]Rule, 0, len(rules))
	for _, rule := range rules {
		ret = append(ret, *rule)
	}
	return ret
}

// ClearRules clears all the rules in flow module.
func ClearRules() error {
	_, err := LoadRules(nil)
	return err
}

func rulesFrom(m TrafficControllerMap) []*Rule {
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

func generateStatFor(rule *Rule) (*standaloneStatistic, error) {
	if !rule.needStatistic() {
		return nopStat, nil
	}

	intervalInMs := rule.StatIntervalInMs

	var retStat standaloneStatistic

	var resNode *stat.ResourceNode
	if rule.RelationStrategy == AssociatedResource {
		// use associated statistic
		resNode = stat.GetOrCreateResourceNode(rule.RefResource, base.ResTypeCommon)
	} else {
		resNode = stat.GetOrCreateResourceNode(rule.Resource, base.ResTypeCommon)
	}
	if intervalInMs == 0 || intervalInMs == config.MetricStatisticIntervalMs() {
		// default case, use the resource's default statistic
		readStat := resNode.DefaultMetric()
		retStat.reuseResourceStat = true
		retStat.readOnlyMetric = readStat
		retStat.writeOnlyMetric = nil
		return &retStat, nil
	}

	sampleCount := uint32(0)
	//calculate the sample count
	if intervalInMs > config.GlobalStatisticIntervalMsTotal() {
		sampleCount = 1
	} else if intervalInMs < config.GlobalStatisticBucketLengthInMs() {
		sampleCount = 1
	} else {
		if intervalInMs%config.GlobalStatisticBucketLengthInMs() == 0 {
			sampleCount = intervalInMs / config.GlobalStatisticBucketLengthInMs()
		} else {
			sampleCount = 1
		}
	}
	err := base.CheckValidityForReuseStatistic(sampleCount, intervalInMs, config.GlobalStatisticSampleCountTotal(), config.GlobalStatisticIntervalMsTotal())
	if err == nil {
		// global statistic reusable
		readStat, e := resNode.GenerateReadStat(sampleCount, intervalInMs)
		if e != nil {
			return nil, e
		}
		retStat.reuseResourceStat = true
		retStat.readOnlyMetric = readStat
		retStat.writeOnlyMetric = nil
		return &retStat, nil
	} else if err == base.GlobalStatisticNonReusableError {
		logging.Info("[FlowRuleManager] Flow rule couldn't reuse global statistic and will generate independent statistic", "rule", rule)
		retStat.reuseResourceStat = false
		realLeapArray := sbase.NewBucketLeapArray(sampleCount, intervalInMs)
		metricStat, e := sbase.NewSlidingWindowMetric(sampleCount, intervalInMs, realLeapArray)
		if e != nil {
			return nil, errors.Errorf("fail to generate statistic for warm up rule: %+v, err: %+v", rule, e)
		}
		retStat.readOnlyMetric = metricStat
		retStat.writeOnlyMetric = realLeapArray
		return &retStat, nil
	}
	return nil, errors.Wrapf(err, "fail to new standalone statistic because of invalid StatIntervalInMs in flow.Rule, StatIntervalInMs: %d", intervalInMs)
}

// SetTrafficShapingGenerator sets the traffic controller generator for the given TokenCalculateStrategy and ControlBehavior.
// Note that modifying the generator of default control strategy is not allowed.
func SetTrafficShapingGenerator(tokenCalculateStrategy TokenCalculateStrategy, controlBehavior ControlBehavior, generator TrafficControllerGenFunc) error {
	if generator == nil {
		return errors.New("nil generator")
	}

	if tokenCalculateStrategy >= Direct && tokenCalculateStrategy <= WarmUp {
		return errors.New("not allowed to replace the generator for default control strategy")
	}
	if controlBehavior >= Reject && controlBehavior <= Throttling {
		return errors.New("not allowed to replace the generator for default control strategy")
	}
	tcMux.Lock()
	defer tcMux.Unlock()

	tcGenFuncMap[trafficControllerGenKey{
		tokenCalculateStrategy: tokenCalculateStrategy,
		controlBehavior:        controlBehavior,
	}] = generator
	return nil
}

func RemoveTrafficShapingGenerator(tokenCalculateStrategy TokenCalculateStrategy, controlBehavior ControlBehavior) error {
	if tokenCalculateStrategy >= Direct && tokenCalculateStrategy <= WarmUp {
		return errors.New("not allowed to replace the generator for default control strategy")
	}
	if controlBehavior >= Reject && controlBehavior <= Throttling {
		return errors.New("not allowed to replace the generator for default control strategy")
	}
	tcMux.Lock()
	defer tcMux.Unlock()

	delete(tcGenFuncMap, trafficControllerGenKey{
		tokenCalculateStrategy: tokenCalculateStrategy,
		controlBehavior:        controlBehavior,
	})
	return nil
}

func getTrafficControllerListFor(name string) []*TrafficShapingController {
	tcMux.RLock()
	defer tcMux.RUnlock()

	return tcMap[name]
}

func calculateReuseIndexFor(r *Rule, oldResTcs []*TrafficShapingController) (equalIdx, reuseStatIdx int) {
	// the index of equivalent rule in old traffic shaping controller slice
	equalIdx = -1
	// the index of statistic reusable rule in old traffic shaping controller slice
	reuseStatIdx = -1

	for idx, oldTc := range oldResTcs {
		oldRule := oldTc.BoundRule()
		if oldRule.isEqualsTo(r) {
			// break if there is equivalent rule
			equalIdx = idx
			break
		}
		// search the index of first stat reusable rule
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

// buildRulesOfRes builds TrafficShapingController slice from rules. the resource of rules must be equals to res
func buildRulesOfRes(res string, rulesOfRes []*Rule, m TrafficControllerMap) []*TrafficShapingController {
	newTcsOfRes := make([]*TrafficShapingController, 0, len(rulesOfRes))
	emptyTcs := make([]*TrafficShapingController, 0, 0)
	for _, rule := range rulesOfRes {
		if res != rule.Resource {
			logging.Error(errors.Errorf("unmatched resource name expect: %s, actual: %s", res, rule.Resource), "Unmatched resource name in flow.buildRulesOfRes()", "rule", rule)
			continue
		}
		oldResTcs, exist := m[res]
		if !exist {
			oldResTcs = emptyTcs
		}

		equalIdx, reuseStatIdx := calculateReuseIndexFor(rule, oldResTcs)

		// First check equals scenario
		if equalIdx >= 0 {
			// reuse the old tc
			equalOldTc := oldResTcs[equalIdx]
			newTcsOfRes = append(newTcsOfRes, equalOldTc)
			// remove old tc from oldResTcs
			m[res] = append(oldResTcs[:equalIdx], oldResTcs[equalIdx+1:]...)
			continue
		}

		generator, supported := tcGenFuncMap[trafficControllerGenKey{
			tokenCalculateStrategy: rule.TokenCalculateStrategy,
			controlBehavior:        rule.ControlBehavior,
		}]
		if !supported || generator == nil {
			logging.Error(errors.New("unsupported flow control strategy"), "Ignoring the rule due to unsupported control behavior in flow.buildRulesOfRes()", "rule", rule)
			continue
		}
		var tc *TrafficShapingController
		var e error
		if reuseStatIdx >= 0 {
			tc, e = generator(rule, &(oldResTcs[reuseStatIdx].boundStat))
		} else {
			tc, e = generator(rule, nil)
		}

		if tc == nil || e != nil {
			logging.Error(errors.New("bad generated traffic controller"), "Ignoring the rule due to bad generated traffic controller in flow.buildRulesOfRes()", "rule", rule)
			continue
		}
		if reuseStatIdx >= 0 {
			// remove old tc from oldResTcs
			m[res] = append(oldResTcs[:reuseStatIdx], oldResTcs[reuseStatIdx+1:]...)
		}
		newTcsOfRes = append(newTcsOfRes, tc)
	}
	return newTcsOfRes
}

// IsValidRule checks whether the given Rule is valid.
func IsValidRule(rule *Rule) error {
	if rule == nil {
		return errors.New("nil Rule")
	}
	if rule.Resource == "" {
		return errors.New("empty Resource")
	}
	if rule.Threshold < 0 {
		return errors.New("negative Threshold")
	}
	if int32(rule.TokenCalculateStrategy) < 0 {
		return errors.New("negative TokenCalculateStrategy")
	}
	if int32(rule.ControlBehavior) < 0 {
		return errors.New("negative ControlBehavior")
	}
	if !(rule.RelationStrategy >= CurrentResource && rule.RelationStrategy <= AssociatedResource) {
		return errors.New("invalid RelationStrategy")
	}
	if rule.RelationStrategy == AssociatedResource && rule.RefResource == "" {
		return errors.New("RefResource must be non empty when RelationStrategy is AssociatedResource")
	}
	if rule.TokenCalculateStrategy == WarmUp {
		if rule.WarmUpPeriodSec <= 0 {
			return errors.New("WarmUpPeriodSec must be great than 0")
		}
		if rule.WarmUpColdFactor == 1 {
			return errors.New("WarmUpColdFactor must be great than 1")
		}
	}
	if rule.StatIntervalInMs > 10*60*1000 {
		logging.Info("StatIntervalInMs is great than 10 minutes, less than 10 minutes is recommended.")
	}
	return nil
}
