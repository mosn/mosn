package system

import (
	"sync"

	"github.com/alibaba/sentinel-golang/logging"

	"github.com/pkg/errors"
)

type RuleMap map[MetricType][]*SystemRule

// const
var (
	logger = logging.GetDefaultLogger()

	ruleMap    = make(RuleMap)
	ruleMapMux = new(sync.RWMutex)
)

// GetRules return all the rules
func GetRules() []*SystemRule {
	ruleMapMux.RLock()
	defer ruleMapMux.RUnlock()

	rules := make([]*SystemRule, 0)
	for _, rs := range ruleMap {
		rules = append(rules, rs...)
	}
	return rules
}

// LoadRules loads given system rules to the rule manager, while all previous rules will be replaced.
func LoadRules(rules []*SystemRule) (bool, error) {
	m := buildRuleMap(rules)

	if err := onRuleUpdate(m); err != nil {
		logger.Errorf("Fail to load rules %+v, err: %+v", rules, err)
		return false, err
	}

	return true, nil
}

// ClearRules clear all the previous rules
func ClearRules() error {
	_, err := LoadRules(nil)
	return err
}

func onRuleUpdate(r RuleMap) error {
	ruleMapMux.Lock()
	defer ruleMapMux.Unlock()

	ruleMap = r
	if len(r) > 0 {
		logger.Infof("[SystemRuleManager] System rules loaded: %v", r)
	} else {
		logger.Info("[SystemRuleManager] System rules were cleared")
	}

	return nil
}

func buildRuleMap(rules []*SystemRule) RuleMap {
	m := make(RuleMap)

	if len(rules) == 0 {
		return m
	}

	for _, rule := range rules {
		if err := IsValidSystemRule(rule); err != nil {
			logger.Warnf("Ignoring invalid system rule: %v, reason: %s", rule, err.Error())
			continue
		}
		rulesOfRes, exists := m[rule.MetricType]
		if !exists {
			m[rule.MetricType] = []*SystemRule{rule}
		} else {
			m[rule.MetricType] = append(rulesOfRes, rule)
		}
	}
	return m
}

// IsValidSystemRule determine the system rule is valid or not
func IsValidSystemRule(rule *SystemRule) error {
	if rule == nil {
		return errors.New("nil SystemRule")
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
