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
	"errors"
	"fmt"
	"os"

	"mosn.io/api"
	"mosn.io/holmes"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/stagemanager"
	"mosn.io/mosn/pkg/types"
)

const logFileName = "holmes.log"

var (
	h *holmes.Holmes
)

/*
 * We intergrate Holmes to MOSN builtin extension.
 * See Holmes: https://github.com/mosn/holmes
 */

type holmesConfig struct {
	Enable           bool // default: false
	DumpPath         string
	BinaryDump       bool
	FullStackDump    bool
	CollectInterval  string
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
	Delay     api.DurationConfig
}

type GoroutineProfileOptions struct {
	*ProfileOptions
	GoroutineTriggerNumMax int // do not profile when goroutine number large than max (to avoid large STW)
}

type ProfileOptions struct {
	Enable      bool
	TriggerMin  int                // not trigger profile when less than min, CPU,memory: percent
	TriggerAbs  int                // always trigger profile when larger than abs, CPU,memory: percent
	TriggerDiff int                // trigger profile when grow than diff, percent
	CoolDown    api.DurationConfig // skip for some time after finished a profile
}

// Init should register to stagemanager Init stage,
// since it must run before the preStart stage (HandleExtendConfig in it)
func Register(_ *v2.MOSNConfig) {
	v2.RegisterParseExtendConfig("holmes", OnHolmesPluginParsed)
}

// Stop should register to stagemanager afterStop stage
func Stop(_ stagemanager.Application) {
	if h != nil {
		h.Stop()
	}
}

// OnHolmesPluginParsed will be called when got the holmes extend config,
// and only start holmes when Enable = true in the config
func OnHolmesPluginParsed(data json.RawMessage) error {
	cfg := &holmesConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("Unmarshal holmes config failed: %v", err)
	}

	// only start holmes when Enable = true
	if !cfg.Enable {
		return nil
	}

	options, err := genHolmesOptions(cfg)
	if err != nil {
		return fmt.Errorf("genHolmesOptions failed: %v", err)
	}
	if h, err = holmes.New(options...); err != nil {
		return fmt.Errorf("new holmes failed: %v", err)
	}
	h.Start()
	return nil
}

func genHolmesOptions(cfg *holmesConfig) ([]holmes.Option, error) {
	var options []holmes.Option
	if !cfg.Enable {
		return options, nil
	}

	// use MosnBasePath/holmes as default dump path
	dumpPath := cfg.DumpPath
	if dumpPath == "" {
		dumpPath = types.MosnBasePath + string(os.PathSeparator) + "holmes"
	}
	options = append(options, holmes.WithDumpPath(dumpPath))

	// use dump path / holmes.log as logfile
	logFile := dumpPath + string(os.PathSeparator) + logFileName
	logger, err := log.GetOrCreateDefaultErrorLogger(logFile, log.INFO)
	if err != nil {
		return nil, err
	}
	options = append(options, holmes.WithLogger(logger))

	if cfg.BinaryDump {
		options = append(options, holmes.WithBinaryDump())
	} else {
		options = append(options, holmes.WithTextDump())
	}

	options = append(options, holmes.WithFullStack(cfg.FullStackDump))

	if cfg.CollectInterval != "" {
		options = append(options, holmes.WithCollectInterval(cfg.CollectInterval))
	}

	if cfg.CPUMax > 0 {
		options = append(options, holmes.WithCPUMax(cfg.CPUMax))
	}

	if c := cfg.CPUProfile; c.Enable {
		opt := holmes.WithCPUDump(c.TriggerMin, c.TriggerDiff, c.TriggerAbs, c.CoolDown.Duration)
		options = append(options, opt)
	}

	if c := cfg.MemProfile; c.Enable {
		opt := holmes.WithMemDump(c.TriggerMin, c.TriggerDiff, c.TriggerAbs, c.CoolDown.Duration)
		options = append(options, opt)
	}

	if c := cfg.GCHeapProfile; c.Enable {
		opt := holmes.WithGCHeapDump(c.TriggerMin, c.TriggerDiff, c.TriggerAbs, c.CoolDown.Duration)
		options = append(options, opt)
	}

	if c := cfg.GoroutineProfile; c.Enable {
		opt := holmes.WithGoroutineDump(c.TriggerMin, c.TriggerDiff, c.TriggerAbs, c.GoroutineTriggerNumMax, c.CoolDown.Duration)
		options = append(options, opt)
	}

	if c := cfg.ThreadProfile; c.Enable {
		opt := holmes.WithThreadDump(c.TriggerMin, c.TriggerDiff, c.TriggerAbs, c.CoolDown.Duration)
		options = append(options, opt)
	}

	if c := cfg.ShrinkThread; c.Enable {
		opt := holmes.WithShrinkThread(c.Threshold, c.Delay.Duration)
		options = append(options, opt)
	}

	return options, nil
}

// SetOptions change holmes options on fly
func SetOptions(opts []holmes.Option) error {
	if h == nil {
		return errors.New("holmes has not been inited yet")
	}
	return h.Set(opts...)
}
