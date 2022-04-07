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

package stats

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/cel"
	"mosn.io/mosn/pkg/cel/attribute"
	"mosn.io/mosn/pkg/cel/extract"
)

var (
	errValueKind = fmt.Errorf("value type must be int64 or float64 or time.Duration")
	errMatchKind = fmt.Errorf("match type must be of boolean")
)

var compiler = cel.NewExpressionBuilder(extract.Attributemanifest, cel.CompatCEXL)

type Metrics struct {
	overrides   []*override
	definitions []*definition
}

type definition struct {
	Name  string
	Type  MetricType
	Value attribute.Expression
}

type Stat struct {
	MetricType MetricType
	Name       string
	Labels     map[string]string
	Value      int64
}

func newMetrics(confs []*MetricConfig, definitions []*MetricDefinition) (*Metrics, error) {
	s := &Metrics{
		definitions: make([]*definition, 0, len(definitions)),
		overrides:   make([]*override, 0, len(confs)),
	}
	for _, d := range definitions {
		expr, kind, err := compiler.Compile(d.Value)
		if err != nil {
			return nil, err
		}
		if kind != attribute.INT64 && kind != attribute.DOUBLE && kind != attribute.DURATION {
			return nil, errValueKind
		}
		s.definitions = append(s.definitions, &definition{
			Name:  d.Name,
			Type:  d.Type,
			Value: expr,
		})
	}

	for _, conf := range confs {
		o := &override{
			Name:         conf.Name,
			TagsToRemove: conf.TagsToRemove,
		}
		dimensions := map[string]attribute.Expression{}
		for key, temp := range conf.Dimensions {
			expr, _, err := compiler.Compile(temp)
			if err != nil {
				return nil, err
			}
			dimensions[key] = expr
		}
		o.Dimensions = dimensions

		if conf.Match != "" {
			expr, kind, err := compiler.Compile(conf.Match)
			if err != nil {
				return nil, err
			}
			if kind != attribute.BOOL {
				return nil, errMatchKind
			}
			o.Match = expr
		}

		s.overrides = append(s.overrides, o)
	}
	return s, nil
}

func (m *Metrics) Stat(bag attribute.Bag) ([]*Stat, error) {
	ss := make([]*Stat, 0, len(m.definitions))
	for _, def := range m.definitions {
		value, err := def.Value.Evaluate(bag)
		if err != nil {
			return nil, err
		}
		var val int64
		switch t := value.(type) {
		case int64:
			val = t
		case float64:
			val = int64(t)
		case time.Duration:
			val = int64(t / time.Millisecond)
		default:
			return nil, errValueKind
		}
		s := &Stat{
			MetricType: def.Type,
			Name:       def.Name,
			Labels:     map[string]string{},
			Value:      val,
		}

		for _, over := range m.overrides {
			err = over.Handle(bag, s)
			if err != nil {
				return nil, err
			}
		}

		ss = append(ss, s)
	}
	return ss, nil
}

type override struct {
	Name         string
	Match        attribute.Expression
	Dimensions   map[string]attribute.Expression
	TagsToRemove []string
}

func (o *override) Handle(bag attribute.Bag, stat *Stat) error {
	if o.Name != "" && o.Name != stat.Name {
		return nil
	}
	if o.Match != nil {
		b, err := o.Match.Evaluate(bag)
		if err != nil {
			return err
		}
		switch t := b.(type) {
		case bool:
			if !t {
				return nil
			}
		default:
			return errMatchKind
		}
	}

	for key, dimension := range o.Dimensions {
		v, err := dimension.Evaluate(bag)
		if err != nil {
			return err
		}
		var str string
		switch s := v.(type) {
		case string:
			str = s
		case api.HeaderMap:
			strs := []string{}
			s.Range(func(key, value string) bool {
				strs = append(strs, strings.Join([]string{key, value}, " => "))
				return true
			})
			// Implementation of api.HeaderMap is uncertain, the order of range is not always stable,
			// so it needs to be sorted.
			sort.Strings(strs)
			str = strings.Join(strs, ", ")
		default:
			str = fmt.Sprint(s)
		}
		stat.Labels[key] = str
	}

	for _, r := range o.TagsToRemove {
		delete(stat.Labels, r)
	}
	return nil
}
