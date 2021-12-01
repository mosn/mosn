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
	"os"
	"os/signal"
	"syscall"

	stm "mosn.io/mosn/pkg/stagemanager"
	"mosn.io/pkg/log"
	"mosn.io/pkg/utils"
)

func init() {
	catchSignals()
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
	switch sig {
	case syscall.SIGUSR1:
		// reopen log
		log.Reopen()
	case syscall.SIGHUP:
		stm.Notice(stm.Active_Reconfiguring)
	case syscall.SIGQUIT:
		// stop mosn gracefully
		stm.Notice(stm.GracefulQuitting)
	default:
		// stop mosn
		stm.Notice(stm.Quitting)
	}
}
