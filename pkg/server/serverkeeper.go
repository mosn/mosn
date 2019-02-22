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

package server

import (
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/metrics"
	"path/filepath"
)

func init() {
	catchSignals()
}

var (
	oldPid                int
	pidFile               string
	onProcessExit         []func()
	GracefulTimeout       = time.Second * 30 //default 30s
	shutdownCallbacksOnce sync.Once
	shutdownCallbacks     []func() error
)

func SetPid(pid string) {
	if pid == "" {
		pidFile = MosnLogBasePath + string(os.PathSeparator) + MosnPidFileName
	} else {
		if err := os.MkdirAll(filepath.Dir(pid), 0644); err != nil {
			pidFile = MosnLogBasePath + string(os.PathSeparator) + MosnPidFileName
		} else {
			pidFile = pid
		}
	}
	storeOldPid()
}

func storeOldPid() {
	if b, err := ioutil.ReadFile(pidFile); err == nil {
		// delete "\n"
		oldPid, _ = strconv.Atoi(string(b[0 : len(b)-1]))
	}
}

func WritePidFile() error {
	pid := []byte(strconv.Itoa(os.Getpid()) + "\n")

	os.MkdirAll(MosnBasePath, 0644)

	return ioutil.WriteFile(pidFile, pid, 0644)
}

func catchSignals() {
	catchSignalsCrossPlatform()
	catchSignalsPosix()
}

func catchSignalsCrossPlatform() {
	go func() {
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
				exitCode := executeShutdownCallbacks("SIGTERM")
				for _, f := range onProcessExit {
					f() // only perform important cleanup actions
				}
				Stop()

				os.Exit(exitCode)
			case syscall.SIGUSR1:
				// reopen
				log.Reopen()
			case syscall.SIGHUP:
				// stop stoppable before reload
				stopStoppable()
				// reload
				reconfigure(true)
			case syscall.SIGUSR2:
				// stop stoppable before reload
				stopStoppable()
				reconfigure(false)
			}
		}
	}()
}

func catchSignalsPosix() {
	go func() {
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
				os.Exit(executeShutdownCallbacks("SIGINT"))
			}()
		}
	}()
}

func executeShutdownCallbacks(signame string) (exitCode int) {
	shutdownCallbacksOnce.Do(func() {
		var errs []error

		for _, cb := range shutdownCallbacks {
			errs = append(errs, cb())
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

func OnProcessShutDown(cb func() error) {
	shutdownCallbacks = append(shutdownCallbacks, cb)
}

func startNewMosn() error {
	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: append([]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}),
	}

	// Fork exec the new version of your server
	fork, err := syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		log.DefaultLogger.Errorf("Fail to fork %v", err)
		return err
	}

	log.DefaultLogger.Infof("SIGHUP received: fork-exec to %d", fork)
	return nil
}

func reconfigure(new bool) {
	if new {
		err := startNewMosn()
		if err != nil {
			return
		}
	}

	// transfer listen fd
	if err := sendInheritListeners(); err != nil {
		return
	}

	// Wait for new mosn start
	time.Sleep(3 * time.Second)

	// Stop accepting requests
	StopAccept()

	// Wait for all connections to be finished
	WaitConnectionsDone(GracefulTimeout)
	// Transfer metrcis data, non-block
	metrics.TransferMetrics(false, 0)
	log.DefaultLogger.Infof("process %d gracefully shutdown", os.Getpid())

	// Stop the old server, all the connections have been closed and the new one is running
	os.Exit(0)
}
