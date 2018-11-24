package router

import (
	"reflect"
	"testing"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func TestDirectResponse(t *testing.T) {
	routeConfigStr := `{
		"match": {
			"prefix": "/"
		},
		"route": {
			"cluster_name":"testcluster"
		},
		"direct_response": {
			"status": 500,
			"body": "test"
		}
	}`
	routeCfg := &v2.Router{}
	if err := json.Unmarshal([]byte(routeConfigStr), routeCfg); err != nil {
		t.Fatal("unmarshal config to router failed, ", err)
	}
	rule, _ := NewRouteRuleImplBase(nil, routeCfg)
	if rule.DirectResponseRule() == nil {
		t.Fatal("rule have no direct response rule")
	}
	dr := rule.DirectResponseRule()
	if dr.StatusCode() != 500 || dr.Body() != "test" {
		t.Error("direct response rule is not exepcted")
	}
	// Test No Direct response by default
	noDirectCfgStr := `{
		"match": {
			"prefix": "/"
		},
		"route": {
			 "cluster_name":"testcluster"
		}
	}`
	noDirectCfg := &v2.Router{}
	if err := json.Unmarshal([]byte(noDirectCfgStr), noDirectCfg); err != nil {
		t.Fatal("unmarshal config to router failed, ", err)
	}
	noDirectRule, _ := NewRouteRuleImplBase(nil, noDirectCfg)
	if !reflect.ValueOf(noDirectRule.DirectResponseRule()).IsNil() {
		t.Error("expected a nil resposne rule, but not", noDirectRule.DirectResponseRule())
	}
}
