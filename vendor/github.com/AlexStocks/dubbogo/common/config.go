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

package common

import (
	"fmt"
)

type ApplicationConfig struct {
	// 组织名(BU或部门)
	Organization string
	// 应用名称
	Name string
	// 模块名称
	Module string
	// 模块版本
	Version string
	// 应用负责人
	Owner string
}

func (c *ApplicationConfig) ToString() string {
	return fmt.Sprintf("ApplicationConfig is {name:%s, version:%s, owner:%s, module:%s, organization:%s}",
		c.Name, c.Version, c.Owner, c.Module, c.Organization)
}
