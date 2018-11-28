// Copyright (c) 2015 Asim Aslam.
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
	"errors"
)

import (
	"github.com/AlexStocks/dubbogo/registry"
)

type NewSelector func(...Option) Selector

// Selector builds on the registry as a mechanism to pick nodes.
// This allows host pools and other things to be built using
// various algorithms.
type Selector interface {
	Options() Options
	// Select returns a function which should return the next node
	Select(conf registry.ServiceConfigIf) (Next, error)
	// Close renders the selector unusable
	Close() error
	// Name of the selector
	String() string
}

// Next is a function that returns the next node
// based on the selector's strategy
type Next func(ID int64) (*registry.ServiceURL, error)

var (
	ErrNotFound              = errors.New("not found")
	ErrNoneAvailable         = errors.New("none available")
	ErrRunOutAllServiceNodes = errors.New("has used out all provider nodes")
)
