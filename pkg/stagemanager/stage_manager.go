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

/*
 * NOTICE:
 * 1. package level public function exposed for for all modules.
 * 2. stagemanager's public method designed to only used in glue level like in the control.go.
 * since we don't want to expose all of them to all modules at the beginning.
 */

package stagemanager

import (
	"os"
	"syscall"
	"time"

	"github.com/urfave/cli"
	"mosn.io/mosn/pkg/admin/store"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/server/pid"
	"mosn.io/mosn/pkg/types"
	logger "mosn.io/pkg/log"
)

type State int

const (
	Init State = iota
	Running
	Active_Reconfiguring
	Passive_Reconfiguring
	GracefulQuitting
	Quitting
)

type Mosn interface {
	InitMosn(*v2.MOSNConfig)
	Start()
	Wait()
	Finish()
	Close()
}

var (
	stm StageManager = StageManager{
		state:          Init,
		data:           Data{},
		started:        false,
		paramsStages:   []func(*cli.Context){},
		initStages:     []func(*v2.MOSNConfig){},
		preStartStages: []func(Mosn){},
		startupStages:  []func(Mosn){},
	}
)

// Data contains objects used in stages
type Data struct {
	// ctx contains the start parameters
	ctx *cli.Context
	// config path represents the config file path
	// mosn create basic config from it and if auto config dump is setted
	// new config data will write into this path
	configPath string
	// mosn basic config, created after parameters parsed stage
	config *v2.MOSNConfig
	// mosn struct contains some basic structures.
	mosn Mosn
}

// StageManager is a stage manager that controls startup stages running.
// We divide the startup into six stages:
// 1. The parameters parsed stage. In this stage, parse different parameters from cli.Context,
// and finally call config load to create a MOSNConfig.
// 2. The initialize stage. In this stage, do some init actions based on config, and finally create a MOSN object.
// 3. The pre-startup stage. In this stage, creates some basic instance from MOSN object.
// 4. The startup stage. In this stage, do some startup actions such as connections transfer for smooth upgrade and so on.
// 5. The after-start stage. In this stage, do some other init actions after startup.
// 6. The pre-stop stage. In this stage, do graceful shutdown actions before mosn closed.
// 7. The after-stop stage. In this stage, do some clean up actions after mosn closed.
// The difference between pre-startup stage and startup stage is that startup stage has already accomplished the resources
// that used to startup mosn.
type StageManager struct {
	state                   State
	data                    Data
	started                 bool
	paramsStages            []func(*cli.Context)
	initStages              []func(*v2.MOSNConfig)
	preStartStages          []func(Mosn)
	startupStages           []func(Mosn)
	afterStartStages        []func(Mosn)
	preStopStages           []func(Mosn)
	afterStopStages         []func(Mosn)
	onStateChangedCallbacks []func(State)
	reconfigureHandler      func() error
}

func InitStageManager(ctx *cli.Context, path string, mosn Mosn) *StageManager {
	stm.data.configPath = path
	stm.data.ctx = ctx
	stm.data.mosn = mosn

	RegisterOnStateChanged(func(s State) {
		metrics.SetStateCode(int64(s))
	})

	return &stm
}

func (stm *StageManager) AppendParamsParsedStage(f func(*cli.Context)) *StageManager {
	if f == nil || stm.started {
		log.StartLogger.Errorf("[stage] invalid stage function or mosn is already started")
		return stm
	}
	stm.paramsStages = append(stm.paramsStages, f)
	return stm
}

func (stm *StageManager) runParamsParsedStage() time.Duration {
	st := time.Now()
	for _, f := range stm.paramsStages {
		f(stm.data.ctx)
	}
	// after all registered stages are completed, call the last process: load config
	stm.data.config = configmanager.Load(stm.data.configPath)
	return time.Since(st)
}

func (stm *StageManager) AppendInitStage(f func(*v2.MOSNConfig)) *StageManager {
	if f == nil || stm.started {
		log.StartLogger.Errorf("[stage] invalid stage function or mosn is already started")
		return stm
	}
	stm.initStages = append(stm.initStages, f)
	return stm
}

func (stm *StageManager) runInitStage() time.Duration {
	st := time.Now()
	for _, f := range stm.initStages {
		f(stm.data.config)
	}
	// after all registered stages are completed, call the last process: init mosn
	stm.data.mosn.InitMosn(stm.data.config)
	return time.Since(st)
}

func (stm *StageManager) AppendPreStartStage(f func(Mosn)) *StageManager {
	if f == nil || stm.started {
		log.StartLogger.Errorf("[stage] invalid stage function or mosn is already started")
		return stm
	}
	stm.preStartStages = append(stm.preStartStages, f)
	return stm
}

func (stm *StageManager) runPreStartStage() time.Duration {
	st := time.Now()
	for _, f := range stm.preStartStages {
		f(stm.data.mosn)
	}
	return time.Since(st)
}

func (stm *StageManager) AppendStartStage(f func(Mosn)) *StageManager {
	if f == nil || stm.started {
		log.StartLogger.Errorf("[stage] invalid stage function or mosn is already started")
		return stm
	}
	stm.startupStages = append(stm.startupStages, f)
	return stm
}

func (stm *StageManager) runStartStage() time.Duration {
	st := time.Now()
	for _, f := range stm.startupStages {
		f(stm.data.mosn)
	}
	// start mosn after all start stages finished
	stm.data.mosn.Start()
	return time.Since(st)
}

func (stm *StageManager) AppendAfterStartStage(f func(Mosn)) *StageManager {
	if f == nil || stm.started {
		log.StartLogger.Errorf("[stage] invalid stage function or mosn is already started")
		return stm
	}
	stm.afterStartStages = append(stm.afterStartStages, f)
	return stm
}

func (stm *StageManager) runAfterStartStage() time.Duration {
	st := time.Now()
	for _, f := range stm.afterStartStages {
		f(stm.data.mosn)
	}
	return time.Since(st)
}

// Run blocks until the mosn is closed
func (stm *StageManager) Run() {
	// 0: mark already started
	stm.started = true
	// 1: parser params
	elapsed := stm.runParamsParsedStage()
	log.StartLogger.Infof("mosn parameters parsed cost: %v", elapsed)
	// 2: init
	elapsed = stm.runInitStage()
	log.StartLogger.Infof("mosn init cost: %v", elapsed)
	// 3: pre start
	elapsed = stm.runPreStartStage()
	log.StartLogger.Infof("mosn prepare to start cost: %v", elapsed)
	// 4: run
	elapsed = stm.runStartStage()
	log.StartLogger.Infof("mosn start cost: %v", elapsed)
	// 5: after start
	elapsed = stm.runAfterStartStage()
	log.StartLogger.Infof("mosn after start cost: %v", elapsed)
}

// WaitFinish waits mosn start finished.
// if Run is not called, return directly
func (stm *StageManager) WaitFinish() {
	if !stm.started {
		return
	}
	stm.data.mosn.Wait()
}

func (stm *StageManager) AppendPreStopStage(f func(Mosn)) *StageManager {
	if f == nil || stm.started {
		log.StartLogger.Errorf("[stage] invalid stage function or mosn is already started")
		return stm
	}
	stm.preStopStages = append(stm.preStopStages, f)
	return stm
}

// gracefull shutdown stage
func (stm *StageManager) runPreStopStage() time.Duration {
	st := time.Now()
	for _, f := range stm.preStopStages {
		f(stm.data.mosn)
	}
	return time.Since(st)
}

func (stm *StageManager) AppendAfterStopStage(f func(Mosn)) *StageManager {
	if f == nil || stm.started {
		log.StartLogger.Errorf("[stage] invalid stage function or mosn is already started")
		return stm
	}
	stm.afterStopStages = append(stm.afterStopStages, f)
	return stm
}

func (stm *StageManager) runAfterStopStage() time.Duration {
	st := time.Now()
	for _, f := range stm.afterStopStages {
		f(stm.data.mosn)
	}
	return time.Since(st)
}

func (stm *StageManager) Stop() {
	if !stm.started {
		return
	}

	var elapsed time.Duration
	if store.GetMosnState() == store.GracefulQuitting {
		elapsed = stm.runPreStopStage()
		log.StartLogger.Infof("mosn pre stop stage cost: %v", elapsed)
	}

	pid.RemovePidFile()

	// Stop mosn
	stm.data.mosn.Close()

	// other cleanup actions
	elapsed = stm.runAfterStopStage()
	log.StartLogger.Infof("mosn after stop stage cost: %v", elapsed)

	logger.CloseAll()
}

func StartNewServer() error {
	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: append([]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}),
	}

	// Fork exec the new version of your server
	fork, err := syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		log.DefaultLogger.Errorf("[server] [reconfigure] Fail to fork %v", err)
		return err
	}

	log.DefaultLogger.Infof("[server] [reconfigure] SIGHUP received: fork-exec to %d", fork)
	return nil
}

func (stm *StageManager) runHupReload() {
	if GetState() != Running {
		log.DefaultLogger.Errorf("[server] [reconfigure] SIGHUP received: current mosn state expected running while got %d", GetState())
		return
	}

	stm.setState(Passive_Reconfiguring)

	// the new started mosn will notice the current old mosn to quit
	// after the new mosn is ready
	err := StartNewServer()
	if err != nil {
		stm.resume()
	}
}

func OnGracefulShutdown(f func()) {
	stm.AppendPreStopStage(func(Mosn) {
		f()
	})
}

func GetState() State {
	return stm.state
}

func (stm *StageManager) setState(s State) {
	stm.state = s
	log.DefaultLogger.Infof("[stm state] state changed to %d", s)
	for _, cb := range stm.onStateChangedCallbacks {
		cb(s)
	}
}

func RegisterOnStateChanged(f func(State)) {
	stm.onStateChangedCallbacks = append(stm.onStateChangedCallbacks, f)
}

func RegsiterReconfigureHandler(f func() error) {
	stm.reconfigureHandler = f
}

// resume the old mosn to running state
func (stm *StageManager) resume() {
	log.DefaultLogger.Infof("[stm state] resume state changed to running")

	stm.setState(Running)
	// Restore PID
	pid.WritePidFile()
}

func (stm *StageManager) runPassiveReconfigure() {
	stm.setState(Passive_Reconfiguring)

	if stm.reconfigureHandler == nil {
		log.DefaultLogger.Alertf(types.ErrorKeyReconfigure, "[old mosn] reconfigureHandler not set yet")
		stm.resume()
		return
	}

	err := stm.reconfigureHandler()
	if err != nil {
		stm.resume()
		return
	}
	stm.Stop()
}

func Notice(s State) {
	switch s {
	case Passive_Reconfiguring:
		stm.runPassiveReconfigure()

	case Active_Reconfiguring:
		stm.runHupReload()

	default:
		stm.setState(s)
		// go back to the main goroutine
		stm.data.mosn.Finish()
	}
}
