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

package misc

import (
	"sync"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/core/log"
	"github.com/alibaba/sentinel-golang/core/stat"
	"github.com/alibaba/sentinel-golang/core/system"
)

var (
	rsSlotChainLock sync.RWMutex
	rsSlotChain     = make(map[string]*base.SlotChain, 8)

	// non-thread safe
	globalStatPrepareSlots = make([]base.StatPrepareSlot, 0, 8)
	globalRuleCheckSlots   = make([]base.RuleCheckSlot, 0, 8)
	globalStatSlot         = make([]base.StatSlot, 0, 8)
)

// non-thread safe
func registerCustomGlobalSlotsToSc(sc *base.SlotChain) {
	if sc == nil {
		return
	}
	for _, s := range globalStatPrepareSlots {
		if base.ValidateStatPrepareSlotNaming(sc, s) {
			sc.AddStatPrepareSlot(s)
		}
	}
	for _, s := range globalRuleCheckSlots {
		if base.ValidateRuleCheckSlotNaming(sc, s) {
			sc.AddRuleCheckSlot(s)
		}
	}
	for _, s := range globalStatSlot {
		if base.ValidateStatSlotNaming(sc, s) {
			sc.AddStatSlot(s)
		}
	}
}

// RegisterGlobalStatPrepareSlot is not thread safe, and user must call RegisterGlobalStatPrepareSlot when initializing sentinel running environment
func RegisterGlobalStatPrepareSlot(slot base.StatPrepareSlot) {
	for _, s := range globalStatPrepareSlots {
		if s.Name() == slot.Name() {
			return
		}
	}
	globalStatPrepareSlots = append(globalStatPrepareSlots, slot)
}

// RegisterGlobalRuleCheckSlot is not thread safe, and user must call RegisterGlobalRuleCheckSlot when initializing sentinel running environment
func RegisterGlobalRuleCheckSlot(slot base.RuleCheckSlot) {
	for _, s := range globalRuleCheckSlots {
		if s.Name() == slot.Name() {
			return
		}
	}
	globalRuleCheckSlots = append(globalRuleCheckSlots, slot)
}

// RegisterGlobalStatSlot is not thread safe, and user must call RegisterGlobalStatSlot when initializing sentinel running environment
func RegisterGlobalStatSlot(slot base.StatSlot) {
	for _, s := range globalStatSlot {
		if s.Name() == slot.Name() {
			return
		}
	}
	globalStatSlot = append(globalStatSlot, slot)
}

func newResourceSlotChain() *base.SlotChain {
	sc := base.NewSlotChain()
	sc.AddStatPrepareSlot(stat.DefaultResourceNodePrepareSlot)

	sc.AddRuleCheckSlot(system.DefaultAdaptiveSlot)

	sc.AddStatSlot(stat.DefaultSlot)
	sc.AddStatSlot(log.DefaultSlot)
	registerCustomGlobalSlotsToSc(sc)
	return sc
}

func RegisterStatPrepareSlotForResource(rsName string, slot base.StatPrepareSlot) {
	rsSlotChainLock.Lock()
	defer rsSlotChainLock.Unlock()

	sc, ok := rsSlotChain[rsName]
	if !ok {
		sc = newResourceSlotChain()
		if base.ValidateStatPrepareSlotNaming(sc, slot) {
			sc.AddStatPrepareSlot(slot)
		}
		rsSlotChain[rsName] = sc
	} else {
		sc.Lock()
		if base.ValidateStatPrepareSlotNaming(sc, slot) {
			sc.AddStatPrepareSlot(slot)
		}
		sc.Unlock()
	}
}

func RegisterRuleCheckSlotForResource(rsName string, slot base.RuleCheckSlot) {
	rsSlotChainLock.Lock()
	defer rsSlotChainLock.Unlock()

	sc, ok := rsSlotChain[rsName]
	if !ok {
		sc = newResourceSlotChain()
		if base.ValidateRuleCheckSlotNaming(sc, slot) {
			sc.AddRuleCheckSlot(slot)
		}
		rsSlotChain[rsName] = sc
	} else {
		sc.Lock()
		if base.ValidateRuleCheckSlotNaming(sc, slot) {
			sc.AddRuleCheckSlot(slot)
		}
		sc.Unlock()
	}

}

func RegisterStatSlotForResource(rsName string, slot base.StatSlot) {
	rsSlotChainLock.Lock()
	defer rsSlotChainLock.Unlock()

	sc, ok := rsSlotChain[rsName]
	if !ok {
		sc = newResourceSlotChain()
		if base.ValidateStatSlotNaming(sc, slot) {
			sc.AddStatSlot(slot)
		}
		rsSlotChain[rsName] = sc
	} else {
		sc.Lock()
		if base.ValidateStatSlotNaming(sc, slot) {
			sc.AddStatSlot(slot)
		}
		sc.Unlock()
	}

}

func GetResourceSlotChain(rsName string) *base.SlotChain {
	rsSlotChainLock.RLock()
	defer rsSlotChainLock.RUnlock()

	sc, ok := rsSlotChain[rsName]
	if !ok {
		return nil
	}

	return sc
}
