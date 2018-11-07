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
	"github.com/alipay/sofa-mosn/pkg/stats"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	writePidFile()

	catchSignals()
}

var (
	pidFile               string
	onProcessExit         []func()
	GracefulTimeout       = time.Second * 30 //default 30s
	shutdownCallbacksOnce sync.Once
	shutdownCallbacks     []func() error
)

func writePidFile() error {
	pidFile = MosnLogBasePath + string(os.PathSeparator) + MosnPidFileName
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
				reconfigure()
			case syscall.SIGUSR2:
				// ignore
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

func reconfigure() {
	// Get socket file descriptor to pass it to fork
	listenerFD := ListListenerFD()
	if len(listenerFD) == 0 {
		log.DefaultLogger.Errorf("no listener fd found")
		return
	}

	// Set a flag for the new process start process
	os.Setenv(types.GracefulRestart, "true")
	os.Setenv(types.InheritFd, strconv.Itoa(len(listenerFD)))

	execSpec := &syscall.ProcAttr{
		Env:   os.Environ(),
		Files: append([]uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd()}, listenerFD...),
	}

	// Fork exec the new version of your server
	fork, err := syscall.ForkExec(os.Args[0], os.Args, execSpec)
	if err != nil {
		log.DefaultLogger.Errorf("Fail to fork %v", err)
		return
	}

	log.DefaultLogger.Infof("SIGHUP received: fork-exec to %d", fork)

	// Wait for new mosn start
	time.Sleep(3 * time.Second)

	// Stop accepting requests
	StopAccept()

	// Wait for all connections to be finished
	WaitConnectionsDone(GracefulTimeout)
	// Transfer metrcis data, non-block
	stats.TransferMetrics(false, 0)
	log.DefaultLogger.Infof("process %d gracefully shutdown", os.Getpid())

	// Stop the old server, all the connections have been closed and the new one is running
	os.Exit(0)
}
