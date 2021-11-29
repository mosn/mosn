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
	"strconv"
	"sync"
	"syscall"

	"mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/log"
	logger "mosn.io/pkg/log"
	"mosn.io/pkg/utils"
)

func init() {
	catchSignals()
}

var (
	pidFile                  string
	finisher                 func(store.State)
	gracefulShutdownRegister func(func())
	hupOnce                  sync.Once
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

func RemovePidFile() {
	if pidFile != "" {
		os.Remove(pidFile)
	}
}

func catchSignals() {
	catchSignalsCrossPlatform()
	catchSignalsPosix()
}

func catchSignalsCrossPlatform() {
	utils.GoWithRecover(func() {
		sigchan := make(chan os.Signal, 1)
		signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGUSR1)

		for sig := range sigchan {
			signalHandler(sig)
		}
	}, nil)
}

func catchSignalsPosix() {
	utils.GoWithRecover(func() {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt)

		sig := <-shutdown
		signalHandler(sig)
	}, nil)
}

func signalHandler(sig os.Signal) {
	log.DefaultLogger.Debugf("signal %s received!", sig)
	// syscall.SIGQUIT, syscall.SIGINT/os.Interrupt or syscall.SIGTERM:
	if finisher == nil {
		log.DefaultLogger.Alertf("keeper.finisher", "finisher is not set yet")
		return
	}
	switch sig {
	case syscall.SIGUSR1:
		// reopen log
		logger.Reopen()
	case syscall.SIGHUP:
		finisher(store.Active_Reconfiguring)
	case syscall.SIGQUIT:
		// stop mosn gracefully
		finisher(store.GracefulQuitting)
	default:
		// stop mosn
		finisher(store.Quitting)
	}
}

// add callback to stagemanager's pre-stop stage
func OnGracefulShutdown(cb func()) {
	if gracefulShutdownRegister == nil {
		log.DefaultLogger.Alertf("keeper.graceful", "gracefulShutdownRegister is not set yet")
		return
	}
	// register cb to stagemanager pre-stop stages
	gracefulShutdownRegister(cb)
}

func RegisterFinisher(f func(store.State)) {
	finisher = f
}

func SetGracefulShutdownRegister(r func(func())) {
	gracefulShutdownRegister = r
}

// start the processes to stop the current mosn
func Shutdown() {
	log.DefaultLogger.Debugf("stop mosn by using a fake INT signal")
	if finisher == nil {
		log.DefaultLogger.Alertf("keeper.finisher", "finisher is not set yet")
		return
	}
	// stop mosn
	finisher(store.Quitting)
}
