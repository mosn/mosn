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
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"net"

	"github.com/alipay/sofa-mosn/pkg/admin/store"
	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/metrics"
	"github.com/alipay/sofa-mosn/pkg/types"
)

func init() {
	catchSignals()
}

var (
	pidFile               string
	onProcessExit         []func()
	GracefulTimeout       = time.Second * 30 //default 30s
	shutdownCallbacksOnce sync.Once
	shutdownCallbacks     []func() error
)

func SetPid(pid string) {
	if pid == "" {
		pidFile = types.MosnPidDefaultFileName
	} else {
		if err := os.MkdirAll(filepath.Dir(pid), 0644); err != nil {
			pidFile = types.MosnPidDefaultFileName
		} else {
			pidFile = pid
		}
	}
	writePidFile()
}

func writePidFile() (err error) {
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
				// reload, fork new mosn
				reconfigure(true)
			case syscall.SIGUSR2:
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

func reconfigure(start bool) {
	if start {
		startNewMosn()
		return
	}
	// set mosn State Reconfiguring
	store.SetMosnState(store.Reconfiguring)
	// if reconfigure failed, set mosn state to Running
	defer store.SetMosnState(store.Running)

	// stop dump config file
	store.DumpLock()
	// if reconfigure failed, enable dump
	defer store.DumpUnlock()

	// transfer listen fd
	var notify net.Conn
	var err error
	var n int
	var buf [1]byte
	if notify, err = sendInheritListeners(); err != nil {
		return
	}

	// Wait new mosn parse configuration
	notify.SetReadDeadline(time.Now().Add(10 * time.Minute))
	n, err = notify.Read(buf[:])
	if n != 1 {
		log.DefaultLogger.Errorf("new mosn start failed")
		return
	}

	// stop other services
	store.StopService()

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

func ReconfigureHandler() {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("transferServer panic %v", r)
		}
	}()
	time.Sleep(time.Second)

	syscall.Unlink(types.ReconfigureDomainSocket)

	l, err := net.Listen("unix", types.ReconfigureDomainSocket)
	if err != nil {
		log.StartLogger.Errorf("reconfigureHandler net listen error: %v", err)
		return
	}
	defer l.Close()

	log.DefaultLogger.Infof("reconfigureHandler start")

	ul := l.(*net.UnixListener)
	for {
		uc, err := ul.AcceptUnix()
		if err != nil {
			log.DefaultLogger.Errorf("reconfigureHandler Accept error :%v", err)
			return
		}
		log.DefaultLogger.Infof("reconfigureHandler Accept")

		_, err = uc.Write([]byte{0})
		if err != nil {
			log.DefaultLogger.Errorf("reconfigureHandler %v", err)
			continue
		}
		uc.Close()

		reconfigure(false)
	}
}

func StopReconfigureHandler() {
	syscall.Unlink(types.ReconfigureDomainSocket)
}

func isReconfigure() bool {
	var unixConn net.Conn
	var err error
	unixConn, err = net.DialTimeout("unix", types.ReconfigureDomainSocket, 1*time.Second)
	if err != nil {
		log.DefaultLogger.Infof("not reconfigure: %v", err)
		return false
	}
	defer unixConn.Close()

	uc := unixConn.(*net.UnixConn)
	buf := make([]byte, 1)
	n, _ := uc.Read(buf)
	if n != 1 {
		return false
	}
	return true
}
