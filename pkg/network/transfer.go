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
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/buffer"
	"github.com/alipay/sofa-mosn/pkg/types"
	"golang.org/x/sys/unix"
)

const (
	transferErr    = 0
	transferNotify = 1
)

// TransferTimeout is the total transfer time
var TransferTimeout = time.Second * 30 //default 30s
var transferDomainSocket = filepath.Dir(os.Args[0]) + string(os.PathSeparator) + "mosn.sock"

// TransferServer is called on new mosn start
func TransferServer(handler types.ConnectionHandler) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("transferServer panic %v", r)
		}
	}()

	if os.Getenv("_MOSN_GRACEFUL_RESTART") != "true" {
		return
	}
	if _, err := os.Stat(transferDomainSocket); err == nil {
		os.Remove(transferDomainSocket)
	}
	l, err := net.Listen("unix", transferDomainSocket)
	if err != nil {
		log.DefaultLogger.Errorf("transfer net listen error %v", err)
		return
	}
	defer l.Close()

	log.DefaultLogger.Infof("TransferServer start")

	var transferMap sync.Map

	go func(handler types.ConnectionHandler) {
		defer func() {
			if r := recover(); r != nil {
				log.DefaultLogger.Errorf("TransferServer panic %v", r)
			}
		}()
		for {
			c, err := l.Accept()
			if err != nil {
				if ope, ok := err.(*net.OpError); ok && (ope.Op == "accept") {
					log.DefaultLogger.Infof("TransferServer listener %s closed", l.Addr())
				} else {
					log.DefaultLogger.Errorf("TransferServer Accept error :%v", err)
				}
				return
			}
			log.DefaultLogger.Infof("transfer Accept")
			go transferHandler(c, handler, &transferMap)
		}
	}(handler)

	select {
	case <-time.After(TransferTimeout*2 + time.Second*10):
		log.DefaultLogger.Infof("TransferServer exit")
		return
	}
}

// transferHandler is called on recv transfer request
func transferHandler(c net.Conn, handler types.ConnectionHandler, transferMap *sync.Map) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("transferHandler panic %v", r)
		}
	}()

	defer c.Close()

	uc, ok := c.(*net.UnixConn)
	if !ok {
		log.DefaultLogger.Errorf("unexpected FileConn type; expected UnixConn, got %T", c)
		return
	}
	// recv type
	conn, err := transferRecvType(uc)
	if err != nil {
		log.DefaultLogger.Errorf("transferRecvType error :%v", err)
		return
	}
	// recv header + buffer
	id, buf, err := transferRecvData(uc)
	if err != nil {
		log.DefaultLogger.Errorf("transferRecvData error :%v", err)
	}

	if conn != nil {
		// transfer read
		connection := transferNewConn(conn, buf, handler, transferMap)
		if connection != nil {
			transferSendID(uc, connection.id)
		} else {
			transferSendID(uc, transferErr)
		}
	} else {
		// transfer write
		connection := transferFindConnection(transferMap, uint64(id))
		if connection == nil {
			log.DefaultLogger.Errorf("transferFindConnection failed")
			return
		}
		err := transferWriteBuffer(connection, buf)
		if err != nil {
			log.DefaultLogger.Errorf("transferWriteBuffer error :%v", err)
			return
		}
	}
}

// old mosn transfer readloop
func transferRead(conn net.Conn, buf types.IoBuffer, logger log.Logger) (uint64, error) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("transferRead panic %v", r)
		}
	}()
	c, err := net.Dial("unix", transferDomainSocket)
	if err != nil {
		logger.Errorf("net Dial unix failed %v", err)
		return transferErr, err
	}

	defer c.Close()

	size := buf.Len()
	tcpconn, ok := conn.(*net.TCPConn)
	if !ok {
		logger.Errorf("unexpected net.Conn type; expected TCPConn, got %T", conn)
		return transferErr, errors.New("unexpected Conn type")
	}
	uc := c.(*net.UnixConn)
	// send type and TCP FD
	err = transferSendType(uc, tcpconn)
	if err != nil {
		logger.Errorf("transferRead failed: %v", err)
		return transferErr, err
	}
	// send header + buffer
	err = transferSendData(uc, 0, buf)
	if err != nil {
		logger.Errorf("transferRead failed: %v", err)
		return transferErr, err
	}
	// recv ID
	id := transferRecvID(uc)
	logger.Infof("TransferRead NewConn id = %d, buffer = %d", id, size)

	return id, nil
}

// old mosn transfer writeloop
func transferWrite(conn *connection, id uint64, logger log.Logger) error {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("transferWrite panic %v", r)
		}
	}()
	c, err := net.Dial("unix", transferDomainSocket)
	if err != nil {
		logger.Errorf("net Dial unix failed %v", err)
		return err
	}
	defer c.Close()

	logger.Infof("TransferWrite id = %d, buffer = %d", id, conn.writeBufLen())
	uc := c.(*net.UnixConn)
	err = transferSendType(uc, nil)
	if err != nil {
		logger.Errorf("transferWrite failed: %v", err)
		return err
	}
	// build net.Buffers to IoBuffer
	buf := transferBuildIoBuffer(conn)
	// send header + buffer
	err = transferSendData(uc, int(id), buf)
	if err != nil {
		logger.Errorf("transferWrite failed: %v", err)
		return err
	}
	return nil
}

func transferBuildIoBuffer(c *connection) types.IoBuffer {
	buf := c.ioBufferPool.Take(c.writeBufLen())
	for _, b := range c.writeBuffers {
         buf.Write(b)
	}
	c.writeBuffers = c.writeBuffers[:0]
	c.ioBuffers = c.ioBuffers[:0]
	return buf
}

func transferWriteBuffer(conn *connection, buf []byte) error {
	iobuf := buffer.NewIoBufferBytes(buf)
	return conn.Write(iobuf)
}

func transferFindConnection(transferMap *sync.Map, id uint64) *connection {
	conn, ok := transferMap.Load(id)
	if !ok {
		return nil
	}
	return conn.(*connection)
}

/**
 *  transfer protocol
 *  header (8 bytes) + data (data length)
 *
 * 0                       4                       8
 * +-----+-----+-----+-----+-----+-----+-----+-----+
 * |      data length      |    connection  ID     |
 * +-----+-----+-----+-----+-----+-----+-----+-----+
 * |                     data                      |
 * +-----+-----+-----+-----+-----+-----+-----+-----+
 *
 **/
func transferBuildHead(size uint32, id uint32) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint32(buf[0:], size)
	binary.BigEndian.PutUint32(buf[4:], id)
	return buf
}

func transferSendHead(uc *net.UnixConn, size uint32, id uint32) error {
	buf := transferBuildHead(size, id)
	return transferSendMsg(uc, buf)
}

/**
 * type (1 bytes)
 *  0 : transfer read and FD
 *  1 : transfer write
 **/
func transferSendType(uc *net.UnixConn, conn *net.TCPConn) error {
	buf := make([]byte, 1)
	// transfer write
	if conn == nil {
		buf[0] = 1
		return transferSendMsg(uc, buf)
	}
	// transfer read, send FD
	return transferSendFD(uc, conn)
}

func transferSendFD(uc *net.UnixConn, conn *net.TCPConn) error {
	buf := make([]byte, 1)
	// transfer read
	buf[0] = 0
	if conn == nil {
		return errors.New("transferSendFD conn is nil")
	}
	f, err := conn.File()
	if err != nil {
		return fmt.Errorf("TCP File failed %v", err)
	}
	defer f.Close()
	rights := syscall.UnixRights(int(f.Fd()))
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

func transferRecvType(uc *net.UnixConn) (net.Conn, error) {
	buf := make([]byte, 1)
	oob := make([]byte, 32)
	_, oobn, _, _, err := uc.ReadMsgUnix(buf, oob)
	if err != nil {
		return nil, fmt.Errorf("ReadMsgUnix error: %v", err)
	}
	// transfer write
	if buf[0] == 1 {
		return nil, nil
	}
	// transfer read, recv FD
	conn, err := transferRecvFD(oob[0:oobn])
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func transferSendData(uc *net.UnixConn, id int, buf types.IoBuffer) error {
	// send header
	err := transferSendHead(uc, uint32(buf.Len()), uint32(id))
	if err != nil {
		return err
	}
	// send read/write buffer
	return transferSendIoBuffer(uc, buf)
}

func transferRecvData(uc *net.UnixConn) (int, []byte, error) {
	// recv header
	size, id, err := transferRecvHead(uc)
	if err != nil {
		return 0, nil, err
	}
	// recv read/write buffer
	buf, err := transferRecvMsg(uc, size)
	if err != nil {
		return 0, nil, err
	}

	return id, buf, nil
}

func transferRecvHead(uc *net.UnixConn) (int, int, error) {
	buf, err := transferRecvMsg(uc, 8)
	if err != nil {
		return 0, 0, fmt.Errorf("ReadMsgUnix error: %v", err)
	}
	size := int(binary.BigEndian.Uint32(buf[0:]))
	id := int(binary.BigEndian.Uint32(buf[4:]))
	return size, id, nil
}

func transferSendIoBuffer(uc *net.UnixConn, buf types.IoBuffer) error {
	if buf.Len() == 0 {
		return nil
	}
	_, err := buf.WriteTo(uc)
	if err != nil {
		return fmt.Errorf("transferSendIobuffer failed: %v", err)
	}
	return nil
}

func transferSendMsg(uc *net.UnixConn, buf []byte) error {
	if len(buf) == 0 {
		return nil
	}
	iobuf := buffer.NewIoBufferBytes(buf)
	err := transferSendIoBuffer(uc, iobuf)
	if err != nil {
		return fmt.Errorf("transferSendMsg failed: %v", err)
	}
	return nil
}

func transferRecvMsg(uc *net.UnixConn, size int) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}
	b := make([]byte, size)
	var n, off int
	var err error
	for {
		n, err = uc.Read(b[off:])
		if err != nil {
			return nil, fmt.Errorf("transferRecvMsg failed: %v", err)
		}
		off += n
		if off == size {
			return b, nil
		}
	}
}

func transferSendID(uc *net.UnixConn, id uint64) error {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(id))
	return transferSendMsg(uc, b)
}

func transferRecvID(uc *net.UnixConn) uint64 {
	b, err := transferRecvMsg(uc, 4)
	if err != nil {
		return transferErr
	}
	return uint64(binary.BigEndian.Uint32(b))
}

func transferNewConn(conn net.Conn, buf []byte, handler types.ConnectionHandler, transferMap *sync.Map) *connection {
	address := conn.LocalAddr().(*net.TCPAddr)
	port := strconv.FormatInt(int64(address.Port), 10)
	ipv4, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:"+port)
	ipv6, _ := net.ResolveTCPAddr("tcp", "[::]:"+port)

	var listener types.Listener
	listener = handler.FindListenerByAddress(ipv6)
	if listener == nil {
		listener = handler.FindListenerByAddress(ipv4)
	}
	if listener == nil {
		log.DefaultLogger.Errorf("Find Listener failed %v", address)
		return nil
	}
	ch := make(chan types.Connection)
	// new connection
	go listener.GetListenerCallbacks().OnAccept(conn, false, nil, ch, buf)
	select {
	// recv connection
	case rch := <-ch:
		conn, ok := rch.(*connection)
		if !ok {
			log.DefaultLogger.Errorf("transfer NewConn failed %v", address)
			return nil
		}
		log.DefaultLogger.Infof("transfer NewConn id: %d", conn.id)
		transferMap.Store(conn.id, conn)
		return conn
	case <-time.After(3000 * time.Millisecond):
		log.DefaultLogger.Errorf("transfer NewConn timeout %v", address)
		return nil
	}
}
