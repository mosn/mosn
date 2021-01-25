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

package network

import (
	"errors"
	"fmt"
	"golang.org/x/sys/unix"
	"net"
	"os"
	"syscall"
)

func transferSendFD(uc *net.UnixConn, file *os.File) error {
	buf := make([]byte, 1)
	// transfer read
	buf[0] = 0
	if file == nil {
		return errors.New("transferSendFD conn is nil")
	}
	defer file.Close()
	rights := syscall.UnixRights(int(file.Fd()))
	n, oobn, err := uc.WriteMsgUnix(buf, rights, nil)
	if err != nil {
		return fmt.Errorf("WriteMsgUnix: %v", err)
	}
	if n != len(buf) || oobn != len(rights) {
		return fmt.Errorf("WriteMsgUnix = %d, %d; want 1, %d", n, oobn, len(rights))
	}
	return nil
}

func transferRecvFD(oob []byte) (net.Conn, error) {
	scms, err := unix.ParseSocketControlMessage(oob)
	if err != nil {
		return nil, fmt.Errorf("ParseSocketControlMessage: %v", err)
	}
	if len(scms) != 1 {
		return nil, fmt.Errorf("expected 1 SocketControlMessage; got scms = %#v", scms)
	}
	scm := scms[0]
	gotFds, err := unix.ParseUnixRights(&scm)
	if err != nil {
		return nil, fmt.Errorf("unix.ParseUnixRights: %v", err)
	}
	if len(gotFds) != 1 {
		return nil, fmt.Errorf("wanted 1 fd; got %#v", gotFds)
	}
	f := os.NewFile(uintptr(gotFds[0]), "fd-from-old")
	defer f.Close()
	conn, err := net.FileConn(f)
	if err != nil {
		return nil, fmt.Errorf("FileConn error :%v", gotFds)
	}
	return conn, nil
}
