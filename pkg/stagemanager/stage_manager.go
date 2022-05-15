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
	"sync"
	"syscall"
	"time"

	"github.com/urfave/cli"

	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/metrics"
	"mosn.io/mosn/pkg/server/pid"
	"mosn.io/mosn/pkg/types"
	logger "mosn.io/pkg/log"
)

type StopAction int

const (
	Stop         StopAction = iota // stop directly
	GracefulStop                   // graceful stop the existing connections
	Reload                         // start a new server
	Upgrade                        // transfer the existing connections to new server
)

type State int

// There are 11 main stages:
// 1. The parameters parsed stage. In this stage, parse different parameters from cli.Context,
// and finally call config load to create a MOSNConfig.
// 2. The initialize stage. In this stage, do some init actions based on config, and finally call Application.Init.
// 3. The pre-startup stage. In this stage, creates some basic instance after executing Application.Init.
// 4. The startup stage. In this stage, do some startup actions such as connections transfer for smooth upgrade and so on.
// 5. The after-start stage. In this stage, do some other init actions after startup.
// 6. The running stage.
// 7. The before-stop stage. In this stage, do actions depend on the "stop action" before stopping service actually,
//    like: unpub from registry or checking the unpub status, make sure it safer for graceful stop.
// 8. The graceful stop stage. In this stage, stop listen and graceful stop the existing connections.
// 9. The stop stage. In this stage, executing Application.Close.
// 10. The after-stop stage. In this stage, do some clean up actions after executing Application.Close
// 11. The stopped stage. everything is closed.
// The difference between pre-startup stage and startup stage is that startup stage has already accomplished the resources
// that used to startup applicaton.
//
// And, there are 2 additional stages:
// 1. Starting a new server. It's for the old server only.
//    The current server will fork a new server when the it receive the HUP signal.
// 2. Upgrading. It's for the old server only too.
//    It means the the new server already started, and the old server is tranferring the config
//    and existing connections to the new server.
const (
	Nil State = iota
	ParamsParsed
	Initing
	PreStart
	Starting
	AfterStart
	Running
	BeforeStop
	GracefulStopping
	Stopping
	AfterStop
	Stopped

	StartingNewServer // start a new server when got HUP signal
	Upgrading         // old server smooth upgrade
)

// the current Application is Mosn,
// we may implement more applications in the future.
type Application interface {
	// inherit config from old server when it exists, otherwise, use the local config
	// init its object members
	Init(*v2.MOSNConfig) error
	// start to work, accepting new connections
	Start()
	// transfer existing connection from old server for smooth upgrade
	InheritConnections() error
	// Shutdown means graceful stop
	Shutdown() error
	// Close means stop working immediately
	Close()
	// IsFromUpgrade application start from upgrade mode,
	// means inherit connections and configuration(if enabled) from old application.
	IsFromUpgrade() bool
}

var (
	stm StageManager = StageManager{
		state:          Nil,
		data:           Data{},
		paramsStages:   []func(*cli.Context){},
		initStages:     []func(*v2.MOSNConfig){},
		preStartStages: []func(Application){},
		startupStages:  []func(Application){},
		newServerC:     make(chan bool, 1),
	}
)

// Data contains objects used in stages
type Data struct {
	// ctx contains the start parameters
	ctx *cli.Context
	// config path represents the config file path,
	// will create basic config from it and if auto config dump is set,
	// new config data will write into this path
	configPath string
	// basic config, created after parameters parsed stage
	config *v2.MOSNConfig
}

// StageManager is used to controls service life stages.
type StageManager struct {
	lock                    sync.Mutex
	state                   State
	exitCode                int
	stopAction              StopAction
	data                    Data
	app                     Application // Application interface
	wg                      sync.WaitGroup
	paramsStages            []func(*cli.Context)
	initStages              []func(*v2.MOSNConfig)
	preStartStages          []func(Application)
	startupStages           []func(Application)
	afterStartStages        []func(Application)
	beforeStopStages        []func(StopAction, Application) error
	gracefulStopStages      []func(Application) error
	afterStopStages         []func(Application)
	onStateChangedCallbacks []func(State)
	upgradeHandler          func() error // old server: send listener/config/old connections to new server
	newServerC              chan bool
}

func InitStageManager(ctx *cli.Context, path string, app Application) *StageManager {
	stm.data.configPath = path
	stm.data.ctx = ctx
	stm.app = app
	stm.state = Nil

	RegisterOnStateChanged(func(s State) {
		metrics.SetStateCode(int64(s))
	})

	return &stm
}

func (stm *StageManager) AppendParamsParsedStage(f func(*cli.Context)) *StageManager {
	if f == nil || stm.state != Nil {
		log.StartLogger.Errorf("[stage] invalid stage function or already started")
		return stm
	}
	stm.paramsStages = append(stm.paramsStages, f)
	return stm
}

func (stm *StageManager) runParamsParsedStage() {
	st := time.Now()
	stm.SetState(ParamsParsed)
	for _, f := range stm.paramsStages {
		f(stm.data.ctx)
	}
	// after all registered stages are completed
	stm.data.config = configmanager.Load(stm.data.configPath)

	log.StartLogger.Infof("parameters parsed stage cost: %v", time.Since(st))
}

// init work base on the local config
func (stm *StageManager) AppendInitStage(f func(*v2.MOSNConfig)) *StageManager {
	if f == nil || stm.state != Nil {
		log.StartLogger.Errorf("[stage] invalid stage function or already started")
		return stm
	}
	stm.initStages = append(stm.initStages, f)
	return stm
}

func (stm *StageManager) runInitStage() {
	st := time.Now()
	stm.SetState(Initing)
	for _, f := range stm.initStages {
		f(stm.data.config)
	}
	// after all registered stages are completed
	if err := stm.app.Init(stm.data.config); err != nil {
		stm.Stop()
	}

	log.StartLogger.Infof("init stage cost: %v", time.Since(st))
}

// runReconfigurationParamsInit to reload mosn configurations from cmd.
func (stm *StageManager) runReconfigurationParamsInit() {
	for _, f := range stm.initStages {
		f(stm.data.config)
	}
}

// more init works after inherit config from old server and new server inited
func (stm *StageManager) AppendPreStartStage(f func(Application)) *StageManager {
	if f == nil || stm.state != Nil {
		log.StartLogger.Errorf("[stage] invalid stage function or already started")
		return stm
	}
	stm.preStartStages = append(stm.preStartStages, f)
	return stm
}

func (stm *StageManager) runPreStartStage() {
	st := time.Now()
	stm.SetState(PreStart)
	for _, f := range stm.preStartStages {
		f(stm.app)
	}
	log.StartLogger.Infof("prepare start stage cost: %v", time.Since(st))
}

// start
func (stm *StageManager) AppendStartStage(f func(Application)) *StageManager {
	if f == nil || stm.state != Nil {
		log.StartLogger.Errorf("[stage] invalid stage function or already started")
		return stm
	}
	stm.startupStages = append(stm.startupStages, f)
	return stm
}

func (stm *StageManager) runStartStage() {
	st := time.Now()
	stm.SetState(Starting)
	for _, f := range stm.startupStages {
		f(stm.app)
	}

	stm.wg.Add(1)
	// start application after all start stages finished
	stm.app.Start()

	// transfer existing connections from old server
	if err := stm.app.InheritConnections(); err != nil {
		// align to the old exit code
		stm.exitCode = 2
		stm.Stop()
	}

	log.StartLogger.Infof("start stage cost: %v", time.Since(st))
}

// after start, already working (accepting request)
func (stm *StageManager) AppendAfterStartStage(f func(Application)) *StageManager {
	if f == nil || stm.state != Nil {
		log.StartLogger.Errorf("[stage] invalid stage function or already started")
		return stm
	}
	stm.afterStartStages = append(stm.afterStartStages, f)
	return stm
}

func (stm *StageManager) runAfterStartStage() {
	st := time.Now()
	stm.SetState(AfterStart)
	for _, f := range stm.afterStartStages {
		f(stm.app)
	}

	log.StartLogger.Infof("after start stage cost: %v", time.Since(st))
}

// Run until the application is started
func (stm *StageManager) Run() {
	// 1: parser params
	stm.runParamsParsedStage()
	// 2: init
	stm.runInitStage()
	// 3: pre start
	stm.runPreStartStage()
	// 4: run
	stm.runStartStage()
	// 5: after start
	stm.runAfterStartStage()

	stm.SetState(Running)
}

// used for the main goroutine wait the finish signal
// if Run is not called, return directly
func (stm *StageManager) WaitFinish() {
	if stm.state == Nil {
		return
	}
	stm.wg.Wait()
}

// graceful stop handlers,
// will exit with non-zero code when a callback handler return error
func (stm *StageManager) AppendGracefulStopStage(f func(Application) error) *StageManager {
	if f == nil || stm.state > Running {
		log.StartLogger.Errorf("[stage] invalid stage function or already stopping")
		return stm
	}
	stm.lock.Lock()
	stm.gracefulStopStages = append(stm.gracefulStopStages, f)
	stm.lock.Unlock()
	return stm
}

// graceful stop tage
func (stm *StageManager) runGracefulStopStage() {
	st := time.Now()
	stm.SetState(GracefulStopping)
	// 1. graceful stop the app firstly
	if err := stm.app.Shutdown(); err != nil {
		log.DefaultLogger.Errorf("failed to graceful stop app: %v", err)
		stm.exitCode = 4
	}
	stm.lock.Lock()
	defer stm.lock.Unlock()
	// 2. run the registered hooks
	for _, f := range stm.gracefulStopStages {
		if err := f(stm.app); err != nil {
			log.DefaultLogger.Errorf("failed to run graceful stop callback: %v", err)
			stm.exitCode = 4
		}
	}

	log.StartLogger.Infof("graceful stop stage cost: %v", time.Since(st))
}

// after application is not working
func (stm *StageManager) AppendAfterStopStage(f func(Application)) *StageManager {
	if f == nil || stm.state != Nil {
		log.StartLogger.Errorf("[stage] invalid stage function or already started")
		return stm
	}
	stm.afterStopStages = append(stm.afterStopStages, f)
	return stm
}

func (stm *StageManager) runAfterStopStage() {
	st := time.Now()
	stm.SetState(AfterStop)
	for _, f := range stm.afterStopStages {
		f(stm.app)
	}

	log.StartLogger.Infof("after stop stage cost: %v", time.Since(st))
}

// exit code:
// 0: normal exit, no error happens
// 1: failed to start
// 4: run before-stop/graceful-stop callback failed
func (stm *StageManager) Stop() {
	if stm.state == Nil {
		return
	}
	preState := stm.state
	// graceful stop the existing connections and requests
	if stm.stopAction == GracefulStop || stm.stopAction == Upgrade {
		stm.runGracefulStopStage()
	}

	stm.SetState(Stopping)

	// do not remove the pid file,
	// since the new started server may have the same pid file
	if preState != Upgrading {
		pid.RemovePidFile()
	}
	// close application
	stm.app.Close()

	// other cleanup actions
	stm.runAfterStopStage()

	if preState != Running {
		log.StartLogger.Errorf("[start] failed to start application at stage: %v", preState)
	}

	logger.CloseAll()
	stm.SetState(Stopped)

	if stm.exitCode != 0 {
		os.Exit(stm.exitCode)
	}

	// main goroutine is not waiting, exit directly
	if preState != Running {
		os.Exit(1)
	}

	// will exit with 0 by default
}

func StartNewServer() error {
	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: append([]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}),
	}

	// Fork exec a new server
	fork, err := syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		log.DefaultLogger.Errorf("[server] Fail to fork %v", err)
		return err
	}

	log.DefaultLogger.Infof("[server] SIGHUP received: fork-exec to %d", fork)
	return nil
}

// start a new server
func (stm *StageManager) runReload() {
	// ignore the HUP signal when it's not running
	if stm.state != Running {
		log.DefaultLogger.Errorf("[server] SIGHUP received: current state expected running while got %d", stm.state)
		return
	}

	stm.SetState(StartingNewServer)

	// the new started server will notice the current old server to stop
	// after the new server is ready
	if err := StartNewServer(); err != nil {
		stm.resume()
	}

	select {
	case <-stm.newServerC:
		// do nothing
	case <-time.After(5 * time.Second):
		// wait max 5 seconds
		// new server not start yet
		log.DefaultLogger.Errorf("[server] still not received message from the new started server after 5 seconds, will resume the current server")
		stm.resume()
	}
}

// will exit with non-zero code when a callback handler return error
func OnGracefulStop(f func() error) {
	stm.AppendGracefulStopStage(func(Application) error {
		return f()
	})
}

// will exit with non-zero code when a callback handler return error
func (stm *StageManager) AppendBeforeStopStage(f func(StopAction, Application) error) *StageManager {
	if f == nil || stm.state > Running {
		log.StartLogger.Errorf("[stage] invalid stage function or already stopping")
		return stm
	}
	stm.lock.Lock()
	stm.beforeStopStages = append(stm.beforeStopStages, f)
	stm.lock.Unlock()
	return stm
}

// will exit with non-zero code when a callback handler return error
func OnBeforeStopStage(f func(StopAction, Application) error) {
	stm.AppendBeforeStopStage(f)
}

func (stm StageManager) runBeforeStopStages() {
	st := time.Now()
	stm.SetState(BeforeStop)
	stm.lock.Lock()
	defer stm.lock.Unlock()
	for _, f := range stm.beforeStopStages {
		if err := f(stm.stopAction, stm.app); err != nil {
			log.DefaultLogger.Errorf("failed to run before-stop callback: %v", err)
			stm.exitCode = 4
		}
	}

	log.StartLogger.Infof("before stop stage cost: %v", time.Since(st))
}

func GetState() State {
	return stm.state
}

// expose this method just make UT easier,
// should not use it directly.
func SetState(s State) {
	stm.SetState(s)
}

// expose this method just make UT easier,
// should not use it directly.
func (stm *StageManager) SetState(s State) {
	stm.state = s
	log.DefaultLogger.Infof("[stagemanager] state changed to %d", s)
	for _, cb := range stm.onStateChangedCallbacks {
		cb(s)
	}
}

func RegisterOnStateChanged(f func(State)) {
	stm.onStateChangedCallbacks = append(stm.onStateChangedCallbacks, f)
}

func RegisterUpgradeHandler(f func() error) {
	stm.upgradeHandler = f
}

// resume the old server to running state
func (stm *StageManager) resume() {
	log.DefaultLogger.Infof("[stagemanager] resume state changed to running")

	stm.SetState(Running)
	// Restore PID
	pid.WritePidFile()
}

// hot upgrade, sending config/existing connections to new server firstly
func (stm *StageManager) runUpgrade() {
	if stm.state == StartingNewServer {
		stm.newServerC <- true
	}
	stm.SetState(Upgrading)

	if stm.upgradeHandler == nil {
		log.DefaultLogger.Alertf(types.ErrorKeyReconfigure, "[old server] upgradeHandler not set yet")
		stm.resume()
		return
	}

	// send to new server, for smooth upgrade
	err := stm.upgradeHandler()
	if err != nil {
		stm.resume()
		return
	}
	// will go back to the main goroutine and stop
	stm.wg.Done()
}

// NoticeStop notices the stop action to stage manager
func NoticeStop(action StopAction) {
	stm.stopAction = action
	stm.runBeforeStopStages()
	switch action {
	case Reload:
		stm.runReload()
	case Upgrade:
		stm.runUpgrade()
	case GracefulStop, Stop:
		if stm.state < AfterStart {
			// stop directly when it hasn't started yet
			stm.Stop()
		} else {
			// will go back to the main goroutine and stop
			stm.wg.Done()
		}
	default:
		// do nothing
	}
}

// run all stages
func (stm *StageManager) RunAll() {
	// start to work
	stm.Run()
	// wait server finished
	stm.WaitFinish()
	// stop working
	stm.Stop()
}

// IsActiveUpgrading just for backward compatible
// means the current application is upgrading from an old application, and not running yet.
// may be inheriting connections or configuration from old one.
func IsActiveUpgrading() bool {
	if stm.app != nil {
		return stm.app.IsFromUpgrade() && stm.state < Running
	}
	return false
}

// StopMosnProcess stops Mosn process via command line,
// and it not support stop on windows.
func (stm *StageManager) StopMosnProcess() (err error) {
	// init
	stm.runParamsParsedStage()
	stm.runReconfigurationParamsInit()

	mosnConfig := stm.data.config

	// finds process and sends SIGINT to mosn process, makes it force exit.
	proc, procid, err := getMosnProcess(mosnConfig)
	if err != nil {
		log.StartLogger.Errorf("[mosn stop] fail to find process(%v), err: %v", procid, err)
		return
	}

	if err = sendSignal2Mosn(proc, procid, syscall.SIGINT); err != nil {
		log.StartLogger.Errorf("[mosn stop] fail to send INT signal to mosn process(%v), err: %v", procid, err)
		time.Sleep(100 * time.Millisecond) // waiting logs output
		return
	}

	// check the process status
	t := time.Now().Add(10 * time.Second)
	cnt := 0
	for {

		if time.Now().After(t) {
			log.StartLogger.Errorf("[mosn stop] mosn process(%v) is still existing after waiting for %v, ignore it and quiting ...", procid, 10*time.Second)
			return
		}

		cnt++
		if cnt%10 == 0 { //log it per second.
			log.StartLogger.Infof("[mosn stop] mosn process(%v)  is still existing, waiting for it quiting", procid)
		}

		if err = proc.Signal(syscall.Signal(0)); err == nil {
			// process alive still.
			time.Sleep(100 * time.Millisecond)
			continue
		}

		log.StartLogger.Infof("[mosn stop] stopped mosn process(%v) successfully.", procid)
		time.Sleep(100 * time.Millisecond) // waiting logs output
		return
	}
}

// ReloadMosnProcess stops Mosn process via command line,
// and it not support reload on windows.
func (stm *StageManager) ReloadMosnProcess() (err error) {
	// init
	stm.runParamsParsedStage()
	stm.runReconfigurationParamsInit()

	mosnConfig := stm.data.config

	log.StartLogger.Infof("[mosn reload] start to reload mosn configurations")
	// get current process info
	var oldProc *os.Process
	var oldProcID int
	if oldProc, oldProcID, err = getMosnProcess(mosnConfig); err != nil {
		log.StartLogger.Errorf("[mosn reload] fail to find process(%v), err: %v", oldProcID, err)
		time.Sleep(100 * time.Millisecond)
		return
	}

	if err = sendSignal2Mosn(oldProc, oldProcID, syscall.SIGHUP); err != nil {
		log.StartLogger.Errorf("[mosn reload] fail to send HUP signal to mosn process(%v), err: %v", oldProcID, err)
		time.Sleep(100 * time.Millisecond) // waiting logs output
		return
	}

	t, cnt, newProcRun, oldProcRun := time.Now().Add(360*time.Second), 0, true, true
	newProc, newProcID := &os.Process{}, int(0)
	for {
		cnt++
		// time-out force exit: both new & old processes are running while run out the time
		if time.Now().After(t) {
			if newProcID == 0 {
				log.StartLogger.Errorf("[mosn reload] fail to find new process(%v), err: %v", mosnConfig.Pid, err)
			} else {
				log.StartLogger.Errorf("[mosn reload] the new process(%v) and old process(%v) are still running after waiting for %v, ignore it and quiting ...", newProcID, oldProcID, t)
			}
			time.Sleep(100 * time.Millisecond) // waiting logs output
			return
		}

		// first time loop in: try to get new process info
		if newProcID == 0 {
			if newProc, newProcID, err = getMosnProcess(mosnConfig); err != nil {
				if cnt > 30 && cnt%100 == 0 { // log warning per 10s, after failed to get new process info for 3s.
					log.StartLogger.Errorf("[mosn reload] failed to get new process(%v)", mosnConfig.Pid)
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}
			log.StartLogger.Infof("[mosn reload] new process(%v) already start, wait the old process(%v) exit.", newProcID, oldProcID)
		}

		// get processes state
		if oldProc.Signal(syscall.Signal(0)) != nil {
			oldProcRun = false
		}

		if newProc.Signal(syscall.Signal(0)) != nil {
			newProcRun = false
		}

		// continue: new run, old run
		if newProcRun && oldProcRun {
			if cnt%10 == 0 { // log per 10s
				log.StartLogger.Infof("[mosn reload] the new process(%v) and old process(%v) are running, wait the old process exit", newProcID, oldProcID)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// success exit: new run, old exit
		if newProcRun && !oldProcRun {
			log.StartLogger.Infof("[mosn reload]reload mosn successfully! The old process(%v) already exited and new process(%v) is running", oldProcID, newProcID)
			time.Sleep(100 * time.Millisecond) // waiting logs output
			return
		}

		// reload failed exit: new exit old run
		if !newProcRun && oldProcRun {
			log.StartLogger.Errorf("[mosn reload]reload mosn failed. The old process(%v) still running and new process exited(%v)", oldProcID, newProcID)
			time.Sleep(100 * time.Millisecond) // waiting logs output
			return
		}
	}
}

func getMosnProcess(cfg *v2.MOSNConfig) (mosnProcess *os.Process, p int, err error) {
	// reads mosn process pid from `mosn.pid` file.
	if p, err = pid.GetPidFrom(cfg.Pid); err != nil {
		log.StartLogger.Errorf("[mosn] fail to get pid: %v", err)
		return
	}

	// finds process and sends SIGINT to mosn process, makes it force exit.
	mosnProcess, err = os.FindProcess(p)
	if err != nil {
		log.StartLogger.Errorf("[mosn] fail to find process(%v), err: %v", p, err)
		return
	}
	return
}

func sendSignal2Mosn(proc *os.Process, p int, sig syscall.Signal) (err error) {

	// check if process is existing.
	err = proc.Signal(syscall.Signal(0))
	if err != nil {
		log.StartLogger.Errorf("[mosn] process(%v) is not existing, err: %v", p, err)
		return
	}

	log.StartLogger.Infof("[mosn] sending signal(%v) to process(%v)", sig.String(), p)
	if err = proc.Signal(sig); err != nil {
		log.StartLogger.Errorf("[mosn] fail to send (%v) to mosn process(%v), err: %v", sig.String(), p, err)
		return
	}
	return
}
