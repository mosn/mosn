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
	"os"
	"runtime/debug"
	"syscall"
	"time"

	"net"

	"mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/stagemanager"
	"mosn.io/mosn/pkg/types"
)

var (
	GracefulTimeout            = time.Second * 30 //default 30s
	enableInheritOldMosnconfig = false
)

func EnableInheritOldMosnconfig(enable bool) {
	enableInheritOldMosnconfig = enable
}

// stage manager will resume the current old mosn when return error
func ReconfigureHandler() error {
	// dump lastest config, and stop DumpConfigHandler()
	configmanager.DumpLock()
	configmanager.DumpConfig()
	// if reconfigure failed, enable DumpConfigHandler()
	defer configmanager.DumpUnlock()

	// transfer listen fd
	var listenSockConn net.Conn
	var err error
	var n int
	var buf [1]byte
	if listenSockConn, err = sendInheritListeners(); err != nil {
		return err
	}

	if enableInheritOldMosnconfig {
		if err = SendInheritConfig(); err != nil {
			listenSockConn.Close()
			log.DefaultLogger.Alertf(types.ErrorKeyReconfigure, "[old mosn] [SendInheritConfig] new mosn start failed")
			return err
		}
	}

	// Wait new mosn parse configuration
	listenSockConn.SetReadDeadline(time.Now().Add(10 * time.Minute))
	n, err = listenSockConn.Read(buf[:])
	if n != 1 {
		log.DefaultLogger.Alertf(types.ErrorKeyReconfigure, "[old mosn] [read ack] new mosn start failed")
		return err
	}

	// ack new mosn
	if _, err := listenSockConn.Write([]byte{0}); err != nil {
		log.DefaultLogger.Alertf(types.ErrorKeyReconfigure, "[old mosn] [write ack] new mosn start failed")
		return err
	}

	// stop other services
	store.StopService()

	// Wait for new mosn start
	time.Sleep(3 * time.Second)

	// Stop accepting new connections & graceful close the existing connections if they supports graceful close.
	shutdownServers()

	// Wait for all connections to be finished
	WaitConnectionsDone(GracefulTimeout)

	log.DefaultLogger.Infof("[server] [reconfigure] process %d gracefully shutdown", os.Getpid())

	// will stop the current old mosn in stage manager
	return nil
}

func ReconfigureListener() {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("[server] [reconfigure] transferServer panic %v\n%s", r, string(debug.Stack()))
		}
	}()
	time.Sleep(time.Second)

	syscall.Unlink(types.ReconfigureDomainSocket)

	l, err := net.Listen("unix", types.ReconfigureDomainSocket)
	if err != nil {
		log.StartLogger.Errorf("[server] [reconfigure] reconfigureHandler net listen error: %v", err)
		return
	}
	defer l.Close()

	log.DefaultLogger.Infof("[server] [reconfigure] reconfigureHandler start")

	ul := l.(*net.UnixListener)
	for {
		uc, err := ul.AcceptUnix()
		if err != nil {
			log.DefaultLogger.Errorf("[server] [reconfigure] reconfigureHandler Accept error :%v", err)
			return
		}
		log.DefaultLogger.Infof("[server] [reconfigure] reconfigureHandler Accept")

		_, err = uc.Write([]byte{0})
		if err != nil {
			log.DefaultLogger.Errorf("[server] [reconfigure] reconfigureHandler %v", err)
			continue
		}
		uc.Close()

		stagemanager.NoticeStop(stagemanager.Upgrade)
	}
}

func StopReconfigureHandler() {
	if stagemanager.GetState() == stagemanager.Upgrading {
		return
	}
	syscall.Unlink(types.ReconfigureDomainSocket)
}

// will notice the old mosn when the reconfigure.sock exists
func IsReconfigure() bool {
	defer func() {
		if r := recover(); r != nil {
			log.StartLogger.Errorf("[server] getInheritListeners panic %v", r)
		}
	}()
	var unixConn net.Conn
	var err error
	unixConn, err = net.DialTimeout("unix", types.ReconfigureDomainSocket, 1*time.Second)
	if err != nil {
		log.DefaultLogger.Infof("[server] [reconfigure] not reconfigure: %v", err)
		return false
	}
	defer unixConn.Close()

	uc := unixConn.(*net.UnixConn)
	buf := make([]byte, 1)
	n, _ := uc.Read(buf)

	return n == 1
}
