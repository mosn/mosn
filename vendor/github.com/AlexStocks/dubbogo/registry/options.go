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

package registry

import (
	"context"
)

import (
	"github.com/AlexStocks/dubbogo/common"
)

type Options struct {
	common.ApplicationConfig
	RegistryConfig

	// Other options for implementations of the interface
	// can be stored in a context
	Context context.Context
}

// Option used to initialise the client
type Option func(*Options)

func ApplicationConf(conf common.ApplicationConfig) Option {
	return func(o *Options) {
		o.ApplicationConfig = conf
	}
}

func RegistryConf(conf RegistryConfig) Option {
	return func(o *Options) {
		o.RegistryConfig = conf
	}
}

func Context(ctx context.Context) Option {
	return func(o *Options) {
		o.Context = ctx
	}
}
