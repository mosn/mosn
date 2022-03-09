/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package holmes

import (
	"encoding/json"
	"github.com/mosn/holmes"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/stagemanager"
	"time"
)

var (
	h *holmes.Holmes
)

type holmesConfig struct {
	Enable           bool
	BinaryDump       bool
	FullStackDump    bool
	CollectInterval  string
	CoolDown         string
	CPUMax           int
	CPUProfile       ProfileOptions
	MemProfile       ProfileOptions
	GCHeapProfile    ProfileOptions
	GoroutineProfile GoroutineProfileOptions
	ThreadProfile    ProfileOptions
	ShrinkThread     ShrinkThreadOptions
}

type ShrinkThreadOptions struct {
	Enable    bool
	Threshold int
	Delay     time.Duration
}

type GoroutineProfileOptions struct {
	*ProfileOptions
	GoroutineTriggerNumMax int // goroutine trigger max in number
}

type ProfileOptions struct {
	Enable      bool
	TriggerMin  int
	TriggerAbs  int
	TriggerDiff int
}

// Init should register to stagemanager Init stage,
// since it must run before the preStart stage (HandleExtendConfig in it)
func Init(_ *v2.MOSNConfig) {
	v2.RegisterParseExtendConfig("holmes", OnHolmesPluginParsed)
}

// Stop should register to stagemanager afterStop stage
func Stop(_ stagemanager.Application) {
	if h != nil {
		h.Stop()
	}
}

func OnHolmesPluginParsed(data json.RawMessage) error {
	cfg := &holmesConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return err
	}

	var err error
	options := genHolmesOptions(cfg)

	if h, err = holmes.New(options...); err != nil {
		return err
	}
	h.Start()
	return nil
}

func genHolmesOptions(cfg *holmesConfig) []holmes.Option {
	var options []holmes.Option
	if cfg.Enable {
		return options
	}
	if cfg.BinaryDump {
		options = append(options, holmes.WithBinaryDump())
	} else {
		options = append(options, holmes.WithTextDump())
	}
	options = append(options, holmes.WithFullStack(cfg.FullStackDump))
	options = append(options, holmes.WithCollectInterval(cfg.CollectInterval))
	options = append(options, holmes.WithCoolDown(cfg.CoolDown))
	if cfg.CPUMax > 0 {
		options = append(options, holmes.WithCPUMax(cfg.CPUMax))
	}
	if c := cfg.CPUProfile; c.Enable {
		opt := holmes.WithCPUDump(c.TriggerMin, c.TriggerDiff, c.TriggerAbs)
		options = append(options, opt)
	}
	if c := cfg.MemProfile; c.Enable {
		opt := holmes.WithMemDump(c.TriggerMin, c.TriggerDiff, c.TriggerAbs)
		options = append(options, opt)
	}
	if c := cfg.GCHeapProfile; c.Enable {
		opt := holmes.WithGCHeapDump(c.TriggerMin, c.TriggerDiff, c.TriggerAbs)
		options = append(options, opt)
	}
	if c := cfg.GoroutineProfile; c.Enable {
		opt := holmes.WithGoroutineDump(c.TriggerMin, c.TriggerDiff, c.TriggerAbs, c.GoroutineTriggerNumMax)
		options = append(options, opt)
	}
	if c := cfg.ThreadProfile; c.Enable {
		opt := holmes.WithThreadDump(c.TriggerMin, c.TriggerDiff, c.TriggerAbs)
		options = append(options, opt)
	}
	if c := cfg.ShrinkThread; c.Enable {
		opt := holmes.WithShrinkThread(c.Enable, c.Threshold, c.Delay)
		options = append(options, opt)
	}
	return options
}

// SetOptions change holmes options on fly
func SetOptions(opts []holmes.Option) {
	h.Set(opts...)
}
