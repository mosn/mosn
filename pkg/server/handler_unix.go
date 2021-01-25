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

package server

import (
	"encoding/json"
	"errors"
	"golang.org/x/sys/unix"
	"io"
	admin "mosn.io/mosn/pkg/admin/store"
	"mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/configmanager"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
	"net"
	"os"
	"syscall"
	"time"
)

func sendInheritListeners() (net.Conn, error) {
	lf := ListListenersFile()
	if lf == nil {
		return nil, errors.New("ListListenersFile() error")
	}

	lsf, lerr := admin.ListServiceListenersFile()
	if lerr != nil {
		return nil, errors.New("ListServiceListenersFile() error")
	}

	var files []*os.File
	files = append(files, lf...)
	files = append(files, lsf...)

	if len(files) > 100 {
		log.DefaultLogger.Errorf("[server] InheritListener fd too many :%d", len(files))
		return nil, errors.New("InheritListeners too many")
	}
	fds := make([]int, len(files))
	for i, f := range files {
		fds[i] = int(f.Fd())
		log.DefaultLogger.Debugf("[server] InheritListener fd: %d", f.Fd())
		defer f.Close()
	}

	var unixConn net.Conn
	var err error
	// retry 10 time
	for i := 0; i < 10; i++ {
		unixConn, err = net.DialTimeout("unix", types.TransferListenDomainSocket, 1*time.Second)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.DefaultLogger.Errorf("[server] sendInheritListeners Dial unix failed %v", err)
		return nil, err
	}

	uc := unixConn.(*net.UnixConn)
	buf := make([]byte, 1)
	rights := syscall.UnixRights(fds...)
	n, oobn, err := uc.WriteMsgUnix(buf, rights, nil)
	if err != nil {
		log.DefaultLogger.Errorf("[server] WriteMsgUnix: %v", err)
		return nil, err
	}
	if n != len(buf) || oobn != len(rights) {
		log.DefaultLogger.Errorf("[server] WriteMsgUnix = %d, %d; want 1, %d", n, oobn, len(rights))
		return nil, err
	}

	return uc, nil
}

// SendInheritConfig send to new mosn using uinx dowmain socket
func SendInheritConfig() error {
	var unixConn net.Conn
	var err error
	// retry 10 time
	for i := 0; i < 10; i++ {
		unixConn, err = net.DialTimeout("unix", types.TransferMosnconfigDomainSocket, 1*time.Second)
		if err == nil {
			break
		}
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		log.DefaultLogger.Errorf("[server] SendInheritConfig Dial unix failed %v", err)
		return err
	}

	configData, err := configmanager.InheritMosnconfig()
	if err != nil {
		return err
	}

	uc := unixConn.(*net.UnixConn)
	defer uc.Close()

	n, err := uc.Write(configData)
	if err != nil {
		log.DefaultLogger.Errorf("[server] Write: %v", err)
		return err
	}
	if n != len(configData) {
		log.DefaultLogger.Errorf("[server] Write = %d, want %d", n, len(configData))
		return errors.New("write mosnconfig data length error")
	}

	return nil
}

func GetInheritListeners() ([]net.Listener, []net.PacketConn, net.Conn, error) {
	defer func() {
		if r := recover(); r != nil {
			log.StartLogger.Errorf("[server] getInheritListeners panic %v", r)
		}
	}()

	if !isReconfigure() {
		return nil, nil, nil, nil
	}

	syscall.Unlink(types.TransferListenDomainSocket)

	l, err := net.Listen("unix", types.TransferListenDomainSocket)
	if err != nil {
		log.StartLogger.Errorf("[server] InheritListeners net listen error: %v", err)
		return nil, nil, nil, err
	}
	defer l.Close()

	log.StartLogger.Infof("[server] Get InheritListeners start")

	ul := l.(*net.UnixListener)
	ul.SetDeadline(time.Now().Add(time.Second * 10))
	uc, err := ul.AcceptUnix()
	if err != nil {
		log.StartLogger.Errorf("[server] InheritListeners Accept error :%v", err)
		return nil, nil, nil, err
	}
	log.StartLogger.Infof("[server] Get InheritListeners Accept")

	buf := make([]byte, 1)
	oob := make([]byte, 1024)
	_, oobn, _, _, err := uc.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, nil, nil, err
	}
	scms, err := unix.ParseSocketControlMessage(oob[0:oobn])
	if err != nil {
		log.StartLogger.Errorf("[server] ParseSocketControlMessage: %v", err)
		return nil, nil, nil, err
	}
	if len(scms) != 1 {
		log.StartLogger.Errorf("[server] expected 1 SocketControlMessage; got scms = %#v", scms)
		return nil, nil, nil, err
	}
	gotFds, err := unix.ParseUnixRights(&scms[0])
	if err != nil {
		log.StartLogger.Errorf("[server] unix.ParseUnixRights: %v", err)
		return nil, nil, nil, err
	}

	var listeners []net.Listener
	var packetConn []net.PacketConn
	for i := 0; i < len(gotFds); i++ {
		fd := uintptr(gotFds[i])
		file := os.NewFile(fd, "")
		if file == nil {
			log.StartLogger.Errorf("[server] create new file from fd %d failed", fd)
			return nil, nil, nil, err
		}
		defer file.Close()

		fileListener, err := net.FileListener(file)
		if err != nil {
			pc, err := net.FilePacketConn(file)
			if err == nil {
				packetConn = append(packetConn, pc)
			} else {

				log.StartLogger.Errorf("[server] recover listener from fd %d failed: %s", fd, err)
				return nil, nil, nil, err
			}
		} else {
			// for tcp or unix listener
			listeners = append(listeners, fileListener)
		}
	}

	return listeners, packetConn, uc, nil
}

func GetInheritConfig() (*v2.MOSNConfig, error) {
	defer func() {
		if r := recover(); r != nil {
			log.StartLogger.Errorf("[server] GetInheritConfig panic %v", r)
		}
	}()

	syscall.Unlink(types.TransferMosnconfigDomainSocket)

	l, err := net.Listen("unix", types.TransferMosnconfigDomainSocket)
	if err != nil {
		log.StartLogger.Errorf("[server] GetInheritConfig net listen error: %v", err)
		return nil, err
	}
	defer l.Close()

	log.StartLogger.Infof("[server] Get GetInheritConfig start")

	ul := l.(*net.UnixListener)
	ul.SetDeadline(time.Now().Add(time.Second * 10))
	uc, err := ul.AcceptUnix()
	if err != nil {
		log.StartLogger.Errorf("[server] GetInheritConfig Accept error :%v", err)
		return nil, err
	}
	defer uc.Close()
	log.StartLogger.Infof("[server] Get GetInheritConfig Accept")
	configData := make([]byte, 0)
	buf := make([]byte, 1024)
	for {
		n, err := uc.Read(buf)
		configData = append(configData, buf[:n]...)
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
	}

	// log.StartLogger.Infof("[server] inherit mosn config data: %v", string(configData))

	oldConfig := &v2.MOSNConfig{}
	err = json.Unmarshal(configData, oldConfig)
	if err != nil {
		return nil, err
	}

	return oldConfig, nil
}
