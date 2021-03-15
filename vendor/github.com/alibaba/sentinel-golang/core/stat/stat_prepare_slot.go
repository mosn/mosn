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

package stat

import (
	"github.com/alibaba/sentinel-golang/core/base"
)

const (
	PrepareSlotName  = "sentinel-core-stat-resource-node-prepare-slot"
	PrepareSlotOrder = 1000
)

var (
	DefaultResourceNodePrepareSlot = &ResourceNodePrepareSlot{}
)

type ResourceNodePrepareSlot struct {
}

func (s *ResourceNodePrepareSlot) Name() string {
	return PrepareSlotName
}

func (s *ResourceNodePrepareSlot) Order() uint32 {
	return PrepareSlotOrder
}

func (s *ResourceNodePrepareSlot) Prepare(ctx *base.EntryContext) {
	node := GetOrCreateResourceNode(ctx.Resource.Name(), ctx.Resource.Classification())
	// Set the resource node to the context.
	ctx.StatNode = node
}
