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
	"github.com/alibaba/sentinel-golang/core/config"
)

type ResourceNode struct {
	BaseStatNode

	resourceName string
	resourceType base.ResourceType
}

// NewResourceNode creates a new resource node with given name and classification.
func NewResourceNode(resourceName string, resourceType base.ResourceType) *ResourceNode {
	return &ResourceNode{
		BaseStatNode: *NewBaseStatNode(config.MetricStatisticSampleCount(), config.MetricStatisticIntervalMs()),
		resourceName: resourceName,
		resourceType: resourceType,
	}
}

func (n *ResourceNode) ResourceType() base.ResourceType {
	return n.resourceType
}

func (n *ResourceNode) ResourceName() string {
	return n.resourceName
}
