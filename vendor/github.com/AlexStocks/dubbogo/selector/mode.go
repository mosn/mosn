// Copyright (c) 2016 ~ 2018, Alex Stocks.
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

package selector

import (
	"math/rand"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/dubbogo/registry"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type ModeFunc func([]*registry.ServiceURL) Next

// Random is a random strategy algorithm for node selection
func random(services []*registry.ServiceURL) Next {
	return func(ID int64) (*registry.ServiceURL, error) {
		if len(services) == 0 {
			return nil, ErrNoneAvailable
		}

		i := ((int64)(rand.Int()) + ID) % (int64)(len(services))
		return services[i], nil
	}
}

// RoundRobin is a roundrobin strategy algorithm for node selection
func roundRobin(services []*registry.ServiceURL) Next {
	var i int64
	var mtx sync.Mutex

	return func(ID int64) (*registry.ServiceURL, error) {
		if len(services) == 0 {
			return nil, ErrNoneAvailable
		}

		mtx.Lock()
		node := services[(ID+i)%int64(len(services))]
		i++
		mtx.Unlock()

		return node, nil
	}
}

//////////////////////////////////////////
// selector mode
//////////////////////////////////////////

// Mode defines the algorithm of selecting a provider from cluster
type Mode int

const (
	SM_BEGIN Mode = iota
	SM_Random
	SM_RoundRobin
	SM_END
)

var selectorModeStrings = [...]string{
	"Begin",
	"Random",
	"RoundRobin",
	"End",
}

func (s Mode) String() string {
	if SM_BEGIN < s && s < SM_END {
		return selectorModeStrings[s]
	}

	return ""
}

var (
	selectorModeFuncs = []ModeFunc{
		SM_BEGIN:      random,
		SM_Random:     random,
		SM_RoundRobin: roundRobin,
		SM_END:        random,
	}
)

func SelectorNext(mode Mode) ModeFunc {
	if mode < SM_BEGIN || SM_END < mode {
		mode = SM_Random
	}

	return selectorModeFuncs[mode]
}
