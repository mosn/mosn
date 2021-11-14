package rules

import (
	"context"
	"mosn.io/mosn/pkg/types"
)

type TransferMatcher interface {
	Matches(ctx context.Context, headers types.HeaderMap) bool
}

type simpleMatcher struct {
}

func (m *simpleMatcher) Matches(ctx context.Context, headers types.HeaderMap) bool {
	return true
}

type TransferRuleMatcher struct {
	MatcherType string `json:"type"`
}

type TransferRuleConfig struct {
	MatcherConfig map[string]interface{} `json:"macther_config"`
	RuleInfo      *RuleInfo              `json:"rule_info"`
}

type MatcherConfig struct {
	Headers   []HeaderMatcher   `json:"headers,omitempty"`
	Variables []VariableMatcher `json:"variables,omitempty"`
}

// HeaderMatcher specifies a set of headers that the rule should match on.
type HeaderMatcher struct {
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
	Regex bool   `json:"regex,omitempty"`
}

// VariableMatcher specifies a set of variables that the rule should match on.
type VariableMatcher struct {
	Name     string `json:"name,omitempty"`
	Value    string `json:"value,omitempty"`
	Regex    bool   `json:"regex,omitempty"`
	Operator string `json:"operator,omitempty"` // support && and || operator
}

type RuleInfo struct {
	UpstreamProtocol    string                 `json:"upstream_protocol"`
	UpstreamSubProtocol string                 `json:"upstream_sub_protocol"`
	Description         string                 `json:"description"`
	Config              map[string]interface{} `json:"config"`
}

func (tf *TransferRuleConfig) Matches(ctx context.Context, headers types.HeaderMap) (*RuleInfo, bool) {
	matcher := GetMatcher()
	result := matcher.Matches(ctx, headers)
	if result {
		return tf.RuleInfo, result
	}
	return nil, false
}
