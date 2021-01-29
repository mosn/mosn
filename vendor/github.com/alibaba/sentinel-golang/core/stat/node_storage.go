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
	"sync"

	"github.com/alibaba/sentinel-golang/core/base"
	"github.com/alibaba/sentinel-golang/logging"
)

type ResourceNodeMap map[string]*ResourceNode

var (
	inboundNode = NewResourceNode(base.TotalInBoundResourceName, base.ResTypeCommon)

	resNodeMap = make(ResourceNodeMap)
	rnsMux     = new(sync.RWMutex)
)

// InboundNode returns the global inbound statistic node.
func InboundNode() *ResourceNode {
	return inboundNode
}

// ResourceNodeList returns the slice of all existing resource nodes.
func ResourceNodeList() []*ResourceNode {
	rnsMux.RLock()
	defer rnsMux.RUnlock()

	list := make([]*ResourceNode, 0, len(resNodeMap))
	for _, v := range resNodeMap {
		list = append(list, v)
	}
	return list
}

func GetResourceNode(resource string) *ResourceNode {
	rnsMux.RLock()
	defer rnsMux.RUnlock()

	return resNodeMap[resource]
}

func GetOrCreateResourceNode(resource string, resourceType base.ResourceType) *ResourceNode {
	node := GetResourceNode(resource)
	if node != nil {
		return node
	}
	rnsMux.Lock()
	defer rnsMux.Unlock()

	node = resNodeMap[resource]
	if node != nil {
		return node
	}

	if len(resNodeMap) >= int(base.DefaultMaxResourceAmount) {
		logging.Warn("[GetOrCreateResourceNode] Resource amount exceeds the threshold", "maxResourceAmount", base.DefaultMaxResourceAmount)
	}
	node = NewResourceNode(resource, resourceType)
	resNodeMap[resource] = node
	return node
}

func ResetResourceNodeMap() {
	rnsMux.Lock()
	defer rnsMux.Unlock()
	resNodeMap = make(ResourceNodeMap)
}
