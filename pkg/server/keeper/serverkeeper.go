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

package keeper

import (
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"sync"
	"syscall"

	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
	"mosn.io/pkg/utils"
)

func init() {
	catchSignals()

	onProcessExit = append(onProcessExit, func() {
		if pidFile != "" {
			os.Remove(pidFile)
		}
	})
}

var (
	pidFile               string
	onProcessExit         []func()
	shutdownCallbacksOnce sync.Once
	shutdownCallbacks     []func() error
	signalCallback        = make(map[syscall.Signal][]func())
)

func SetPid(pid string) {
	if pid == "" {
		pidFile = types.MosnPidDefaultFileName
	} else {
		if err := os.MkdirAll(filepath.Dir(pid), 0755); err != nil {
			pidFile = types.MosnPidDefaultFileName
		} else {
			pidFile = pid
		}
	}
	WritePidFile()
}

func WritePidFile() (err error) {
	pid := []byte(strconv.Itoa(os.Getpid()) + "\n")

	if err = ioutil.WriteFile(pidFile, pid, 0644); err != nil {
		log.DefaultLogger.Errorf("write pid file error: %v", err)
	}
	return err
}

func catchSignals() {
	catchSignalsCrossPlatform()
	catchSignalsPosix()
}

func catchSignalsCrossPlatform() {
	utils.GoWithRecover(func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGHUP,
			syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)

		for sig := range sigchan {
			log.DefaultLogger.Debugf("signal %s received!", sig)
			switch sig {
			case syscall.SIGQUIT:
				// quit
				for _, f := range onProcessExit {
					f() // only perform important cleanup actions
				}
				os.Exit(0)
			case syscall.SIGTERM:
				// stop to quit
				exitCode := ExecuteShutdownCallbacks("SIGTERM")
				for _, f := range onProcessExit {
					f() // only perform important cleanup actions
				}
				//Stop()

				if cbs, ok := signalCallback[syscall.SIGTERM]; ok {
					for _, cb := range cbs {
						cb()
					}
				}

				os.Exit(exitCode)
			case syscall.SIGUSR1:
				// reopen
				log.Reopen()
			case syscall.SIGHUP:

				if cbs, ok := signalCallback[syscall.SIGHUP]; ok {
					for _, cb := range cbs {
						cb()
					}
				}
			case syscall.SIGUSR2:
			}
		}
	}, nil)
}

func catchSignalsPosix() {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.DefaultLogger.Errorf("panic %v\n%s", r, string(debug.Stack()))
			}
		}()
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt)

		for i := 0; true; i++ {
			<-shutdown

			if i > 0 {
				for _, f := range onProcessExit {
					f() // important cleanup actions only
				}
				os.Exit(2)
			}

			// important cleanup actions before shutdown callbacks
			for _, f := range onProcessExit {
				f()
			}

			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.DefaultLogger.Errorf("panic %v\n%s", r, string(debug.Stack()))
					}
				}()
				os.Exit(ExecuteShutdownCallbacks("SIGINT"))
			}()
		}
	}()
}

func ExecuteShutdownCallbacks(signame string) (exitCode int) {
	shutdownCallbacksOnce.Do(func() {
		var errs []error

		for _, cb := range shutdownCallbacks {
			// If the callback is performing normally,
			// err does not need to be saved to prevent
			// the exit code from being non-zero
			if err := cb(); err != nil {
				errs = append(errs, err)
			}
		}

		if len(errs) > 0 {
			for _, err := range errs {
				log.DefaultLogger.Errorf(" %s shutdown: %v", signame, err)
			}
			exitCode = 4
		}
	})

	return
}

func OnProcessExit(cb func()) {
	onProcessExit = append(onProcessExit, cb)
}

func OnProcessShutDown(cb func() error) {
	shutdownCallbacks = append(shutdownCallbacks, cb)
}

// OnProcessShutDownFirst insert the callback func into the header
func OnProcessShutDownFirst(cb func() error) {
	var firstCallbacks []func() error
	firstCallbacks = append(firstCallbacks, cb)
	firstCallbacks = append(firstCallbacks, shutdownCallbacks...)
	// replace current firstCallbacks
	shutdownCallbacks = firstCallbacks
}

func AddSignalCallback(signal syscall.Signal, cb func()) {
	signalCallback[signal] = append(signalCallback[signal], cb)
}
