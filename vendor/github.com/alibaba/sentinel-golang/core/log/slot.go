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

package log

import (
	"github.com/alibaba/sentinel-golang/core/base"
)

const (
	StatSlotName  = "sentinel-core-log-stat-slot"
	StatSlotOrder = 2000
)

var (
	DefaultSlot = &Slot{}
)

type Slot struct {
}

func (s *Slot) Name() string {
	return StatSlotName
}

func (s *Slot) Order() uint32 {
	return StatSlotOrder
}

func (s *Slot) OnEntryPassed(_ *base.EntryContext) {

}

func (s *Slot) OnEntryBlocked(ctx *base.EntryContext, blockError *base.BlockError) {
	// TODO: write sentinel-block.log here
}

func (s *Slot) OnCompleted(_ *base.EntryContext) {

}
