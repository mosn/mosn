// +build darwin linux

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
	"mosn.io/pkg/log"
	"os"
	"os/signal"
	"syscall"
)

func doCatchSignal() {
	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, syscall.SIGTERM, syscall.SIGHUP,
		syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGINT)

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
			executeSignalCallback(syscall.SIGTERM)
			os.Exit(exitCode)
		case syscall.SIGUSR1:
			// reopen
			log.Reopen()
		case syscall.SIGUSR2:
			// do nothing
		case syscall.SIGHUP:
			executeSignalCallback(syscall.SIGHUP)
		case syscall.SIGINT:
			executeSignalCallback(syscall.SIGINT)
		}
	}
}
