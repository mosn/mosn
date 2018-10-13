/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package resource

import (
	"testing"
	"github.com/alipay/sofa-mosn/pkg/protocol"
	"github.com/alipay/sofa-mosn/pkg/filter/stream/commonrule/model"
)

var params = []model.ComparisonCofig{
	{
		Key: "aa",
		Value: "va",
		CompareType: COMPARE_EQUALS,
	},
	{
		Key: "bb",
		Value: "vb",
		CompareType: COMPARE_EQUALS,
	},
}

var params2 = []model.ComparisonCofig{
	{
		Key: "aa",
		Value: "va",
		CompareType: COMPARE_NOT_EQUALS,
	},
	{
		Key: "bb",
		Value: "vb",
		CompareType: COMPARE_NOT_EQUALS,
	},
}

func TestDefaultMatcher_Match(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
		Headers: []model.ComparisonCofig{
		    {
                CompareType:COMPARE_EQUALS,
                Key: protocol.MosnHeaderPathKey,
                Value:"/serverlist/xx.do",
            },
        },
		Params: params,
	}

	headers := protocol.CommonHeader {
		protocol.MosnHeaderPathKey: "/serverlist/xx.do",
		protocol.MosnHeaderQueryStringKey: "aa=va&&bb=vb",
	}

	res := matcher.Match(headers, &resourceConfig)
	if !res {
		t.Errorf("false")
	}
}

func TestDefaultMatcher_Match1(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
        Headers: []model.ComparisonCofig{
            {
                CompareType:COMPARE_EQUALS,
                Key: protocol.MosnHeaderPathKey,
                Value:"/serverlist/xx.do",
            },
        },
		Params: params,
	}

	headers := protocol.CommonHeader {
		protocol.MosnHeaderPathKey: "/serverlist/xx.do",
		protocol.MosnHeaderQueryStringKey: "aa=va&&bb=vb1",
	}

	res := matcher.Match(headers, &resourceConfig)
	if res {
		t.Errorf("false")
	}
}


func TestDefaultMatcher_Match2(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
        Headers: []model.ComparisonCofig{
            {
                CompareType:COMPARE_EQUALS,
                Key: protocol.MosnHeaderPathKey,
                Value:"/serverlist/xx.do",
            },
        },
		Params: params,
	}

	headers := protocol.CommonHeader {
		protocol.MosnHeaderPathKey: "/serverlist/xx.do1",
		protocol.MosnHeaderQueryStringKey: "aa=va&&bb=vb",
	}

	res := matcher.Match(headers, &resourceConfig)
	if res {
		t.Errorf("false")
	}
}


func TestDefaultMatcher_Match3(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
        Headers: []model.ComparisonCofig{
            {
                CompareType:COMPARE_EQUALS,
                Key: protocol.MosnHeaderPathKey,
                Value:"/serverlist/xx.do",
            },
        },
		Params: params,
	}

	headers := protocol.CommonHeader {
		protocol.MosnHeaderPathKey: "/serverlist/xx.do",
		protocol.MosnHeaderQueryStringKey: "aa=va",
	}

	res := matcher.Match(headers, &resourceConfig)
	if res {
		t.Errorf("false")
	}
}


func TestDefaultMatcher_Match4(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
        Headers: []model.ComparisonCofig{
            {
                CompareType:COMPARE_EQUALS,
                Key: protocol.MosnHeaderPathKey,
                Value:"/serverlist/xx.do",
            },
        },
		Params: params,
		ParamsRelation: RELATION_OR,
	}

	headers := protocol.CommonHeader {
		protocol.MosnHeaderPathKey: "/serverlist/xx.do",
		protocol.MosnHeaderQueryStringKey: "aa=va",
	}

	res := matcher.Match(headers, &resourceConfig)
	if !res {
		t.Errorf("false")
	}
}


func TestDefaultMatcher_Match5(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
        Headers: []model.ComparisonCofig{
            {
                CompareType:COMPARE_EQUALS,
                Key: protocol.MosnHeaderPathKey,
                Value:"/serverlist/xx.do",
            },
        },
		Params: params,
		ParamsRelation: RELATION_OR,
	}

	headers := protocol.CommonHeader {
		protocol.MosnHeaderPathKey: "/serverlist/xx.do",
		protocol.MosnHeaderQueryStringKey: "aa=va&&bb=vb1",
	}

	res := matcher.Match(headers, &resourceConfig)
	if !res {
		t.Errorf("false")
	}
}


func TestDefaultMatcher_Match11(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
        Headers: []model.ComparisonCofig{
            {
                CompareType:COMPARE_EQUALS,
                Key: protocol.MosnHeaderPathKey,
                Value:"/serverlist/xx.do",
            },
        },
		Params: params2,
	}

	headers := protocol.CommonHeader {
		protocol.MosnHeaderPathKey: "/serverlist/xx.do",
		protocol.MosnHeaderQueryStringKey: "aa=va&&bb=vb",
	}

	res := matcher.Match(headers, &resourceConfig)
	if res {
		t.Errorf("false")
	}
}


func TestDefaultMatcher_Match12(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
        Headers: []model.ComparisonCofig{
            {
                CompareType:COMPARE_EQUALS,
                Key: protocol.MosnHeaderPathKey,
                Value:"/serverlist/xx.do",
            },
        },
		Params: params2,
	}

	headers := protocol.CommonHeader {
		protocol.MosnHeaderPathKey: "/serverlist/xx.do",
		protocol.MosnHeaderQueryStringKey: "aa=va&&bb=vb1",
	}

	res := matcher.Match(headers, &resourceConfig)
	if res {
		t.Errorf("false")
	}
}


func TestDefaultMatcher_Match13(t *testing.T) {
	matcher := &DefaultMatcher{}
	resourceConfig := model.ResourceConfig{
        Headers: []model.ComparisonCofig{
            {
                CompareType:COMPARE_EQUALS,
                Key: protocol.MosnHeaderPathKey,
                Value:"/serverlist/xx.do",
            },
        },
		Params: params2,
	}

	headers := protocol.CommonHeader {
		protocol.MosnHeaderPathKey: "/serverlist/xx.do",
		protocol.MosnHeaderQueryStringKey: "aa=va1&&bb=vb1",
	}

	res := matcher.Match(headers, &resourceConfig)
	if !res {
		t.Errorf("false")
	}
}