// Copyright 2016 ~ 2018 AlexStocks(https://github.com/AlexStocks).
// All rights reserved.  Use of this source code is
// governed by Apache License 2.0.

// Package gxpool provides a interface for service pool filter
package gxpool

import (
	"time"
)

import (
	"github.com/AlexStocks/goext/context"
	"github.com/AlexStocks/goext/database/filter"
)

const (
	GxfilterDefaultKey     = 0X201804201515
	GxfilterServiceAttrKey = 0X201808162038
)

func WithTTL(t time.Duration) gxfilter.Option {
	return func(o *gxfilter.Options) {
		if o.Context == nil {
			o.Context = gxcontext.NewValuesContext(nil)
		}
		o.Context.Set(GxfilterDefaultKey, t)
	}
}
