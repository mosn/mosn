package flow

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/alibaba/sentinel-golang/logging"
	"github.com/pkg/errors"
)

// const
var (
	logger = logging.GetDefaultLogger()
)

// TrafficControllerGenFunc represents the TrafficShapingController generator function of a specific control behavior.
type TrafficControllerGenFunc func(*FlowRule) *TrafficShapingController

// TrafficControllerMap represents the map storage for TrafficShapingController.
type TrafficControllerMap map[string][]*TrafficShapingController

var (
	tcGenFuncMap = make(map[ControlBehavior]TrafficControllerGenFunc)
	tcMap        = make(TrafficControllerMap)
	tcMux        = new(sync.RWMutex)
)

func init() {
	// Initialize the traffic shaping controller generator map for existing control behaviors.
	tcGenFuncMap[Reject] = func(rule *FlowRule) *TrafficShapingController {
		return NewTrafficShapingController(NewDefaultTrafficShapingCalculator(rule.Count), NewDefaultTrafficShapingChecker(rule.MetricType), rule)
	}
	tcGenFuncMap[Throttling] = func(rule *FlowRule) *TrafficShapingController {
		return NewTrafficShapingController(NewDefaultTrafficShapingCalculator(rule.Count), NewThrottlingChecker(rule.MaxQueueingTimeMs), rule)
	}
}

func logRuleUpdate(m TrafficControllerMap) {
	bs, err := json.Marshal(rulesFrom(m))
	if err != nil {
		logger.Info("[FlowRuleManager] Flow rules loaded")
	} else {
		logger.Infof("[FlowRuleManager] Flow rules loaded: %s", bs)
	}
}

func onRuleUpdate(rules []*FlowRule) (err error) {
	defer func() {
		if r := recover(); r != nil {
			var ok bool
			err, ok = r.(error)
			if !ok {
				err = fmt.Errorf("%v", r)
			}
		}
	}()

	tcMux.Lock()
	defer tcMux.Unlock()

	m := buildFlowMap(rules)

	tcMap = m
	logRuleUpdate(m)

	return nil
}

// LoadRules loads the given flow rules to the rule manager, while all previous rules will be replaced.
func LoadRules(rules []*FlowRule) (bool, error) {
	// TODO: rethink the design
	err := onRuleUpdate(rules)
	return true, err
}

func GetRules() []*FlowRule {
	tcMux.RLock()
	defer tcMux.RUnlock()

	return rulesFrom(tcMap)
}

func ClearRules() error {
	_, err := LoadRules(nil)
	return err
}

func rulesFrom(m TrafficControllerMap) []*FlowRule {
	rules := make([]*FlowRule, 0)
	if len(m) == 0 {
		return rules
	}
	for _, rs := range m {
		if len(rs) == 0 {
			continue
		}
		for _, r := range rs {
			if r != nil && r.Rule() != nil {
				rules = append(rules, r.Rule())
			}
		}
	}
	return rules
}

// SetTrafficShapingGenerator sets the traffic controller generator for the given control behavior.
// Note that modifying the generator of default control behaviors is not allowed.
func SetTrafficShapingGenerator(cb ControlBehavior, generator TrafficControllerGenFunc) error {
	if generator == nil {
		return errors.New("nil generator")
	}
	if cb >= Reject && cb <= WarmUpThrottling {
		return errors.New("not allowed to replace the generator for default control behaviors")
	}
	tcMux.Lock()
	defer tcMux.Unlock()

	tcGenFuncMap[cb] = generator
	return nil
}

func RemoveTrafficShapingGenerator(cb ControlBehavior) error {
	if cb >= Reject && cb <= WarmUpThrottling {
		return errors.New("not allowed to replace the generator for default control behaviors")
	}
	tcMux.Lock()
	defer tcMux.Unlock()

	delete(tcGenFuncMap, cb)
	return nil
}

func getTrafficControllerListFor(name string) []*TrafficShapingController {
	tcMux.RLock()
	defer tcMux.RUnlock()

	return tcMap[name]
}

// NotThreadSafe (should be guarded by the lock)
func buildFlowMap(rules []*FlowRule) TrafficControllerMap {
	m := make(TrafficControllerMap)
	if len(rules) == 0 {
		return m
	}

	for _, rule := range rules {
		if err := IsValidFlowRule(rule); err != nil {
			logger.Warnf("Ignoring invalid flow rule: %v, reason: %s", rule, err.Error())
			continue
		}
		if rule.LimitOrigin == "" {
			rule.LimitOrigin = LimitOriginDefault
		}
		generator, supported := tcGenFuncMap[rule.ControlBehavior]
		if !supported {
			logger.Warnf("Ignoring the rule due to unsupported control behavior: %v", rule)
			continue
		}
		tsc := generator(rule)
		if tsc == nil {
			logger.Warnf("Ignoring the rule due to bad generated traffic controller: %v", rule)
			continue
		}

		rulesOfRes, exists := m[rule.Resource]
		if !exists {
			m[rule.Resource] = []*TrafficShapingController{tsc}
		} else {
			m[rule.Resource] = append(rulesOfRes, tsc)
		}
	}
	return m
}

// IsValidFlowRule checks whether the given FlowRule is valid.
func IsValidFlowRule(rule *FlowRule) error {
	if rule == nil {
		return errors.New("nil FlowRule")
	}
	if rule.Resource == "" {
		return errors.New("empty resource name")
	}
	if rule.Count < 0 {
		return errors.New("negative threshold")
	}
	if rule.MetricType < 0 {
		return errors.New("invalid metric type")
	}
	if rule.RelationStrategy < 0 {
		return errors.New("invalid relation strategy")
	}
	if rule.ControlBehavior < 0 {
		return errors.New("invalid control behavior")
	}

	if rule.RelationStrategy == AssociatedResource && rule.RefResource == "" {
		return errors.New("Bad flow rule: invalid control behavior")
	}
	if err := checkClusterField(rule); err != nil {
		return err
	}

	return checkControlBehaviorField(rule)
}

func checkClusterField(rule *FlowRule) error {
	if rule.ClusterMode && rule.ID <= 0 {
		return errors.New("invalid cluster rule ID")
	}
	return nil
}

func checkControlBehaviorField(rule *FlowRule) error {
	switch rule.ControlBehavior {
	case WarmUp:
		if rule.WarmUpPeriodSec <= 0 {
			return errors.New("invalid warmUpPeriodSec")
		}
		return nil
	case WarmUpThrottling:
		if rule.WarmUpPeriodSec <= 0 {
			return errors.New("invalid warmUpPeriodSec")
		}
		return nil
	default:
	}
	return nil
}
