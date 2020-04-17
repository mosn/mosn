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

package tls

import (
	"bytes"
	"io"
	"net"

	gotls "crypto/tls"
)

// TransferTLSInfo for transfer TLSConn
type TransferTLSInfo struct {
	Vers         uint16
	CipherSuite  uint16
	MasterSecret []byte
	ClientRandom []byte
	ServerRandom []byte
	InSeq        [8]byte
	OutSeq       [8]byte
	RawInput     []byte
	Input        []byte
}

// TransferTLSConn returns Conn by TLSInfo
func TransferTLSConn(conn net.Conn, info *TransferTLSInfo) *Conn {
	c := Server(conn, &Config{})
	c.vers = info.Vers
	c.cipherSuite = info.CipherSuite

	var suite *cipherSuite
	for _, s := range cipherSuites {
		if s.id == c.cipherSuite {
			suite = s
			break
		}
	}

	if suite == nil {
		return nil
	}

	if err := transferEstablishKeys(c, suite, info.MasterSecret, info.ClientRandom, info.ServerRandom); err != nil {
		return nil
	}

	if err := transferChangeCipherSpec(c, info); err != nil {
		return nil
	}

	if info.RawInput != nil {
		c.rawInput = *bytes.NewBuffer(info.RawInput)
		info.RawInput = nil
	}

	if info.Input != nil {
		c.input = *bytes.NewReader(info.Input)
		info.Input = nil
	}

	c.handshakeStatus = 1
	c.info = info

	return c
}

// TransferSetTLSInfo sets TLSInfo
func TransferSetTLSInfo(hs serverHandshakeState) {
	c := hs.c
	info := new(TransferTLSInfo)
	info.Vers = c.vers
	info.CipherSuite = c.cipherSuite
	info.MasterSecret = hs.masterSecret
	if hs.clientHello != nil {
		info.ClientRandom = hs.clientHello.random
	}

	if hs.hello != nil {
		info.ServerRandom = hs.hello.random
	}

	c.info = info
}

func transferEstablishKeys(c *Conn, suite *cipherSuite, masterSecret, clientRandom, serverRandom []byte) error {
	clientMAC, serverMAC, clientKey, serverKey, clientIV, serverIV :=
		keysFromMasterSecret(c.vers, suite, masterSecret, clientRandom, serverRandom, suite.macLen, suite.keyLen, suite.ivLen)

	var clientCipher, serverCipher interface{}
	var clientHash, serverHash macFunction

	if suite.aead == nil {
		clientCipher = suite.cipher(clientKey, clientIV, true /* for reading */)
		clientHash = suite.mac(c.vers, clientMAC)
		serverCipher = suite.cipher(serverKey, serverIV, false /* not for reading */)
		serverHash = suite.mac(c.vers, serverMAC)
	} else {
		clientCipher = suite.aead(clientKey, clientIV)
		serverCipher = suite.aead(serverKey, serverIV)
	}

	c.in.prepareCipherSpec(c.vers, clientCipher, clientHash)
	c.out.prepareCipherSpec(c.vers, serverCipher, serverHash)

	return nil
}

func transferChangeCipherSpec(c *Conn, info *TransferTLSInfo) error {
	if err := c.in.changeCipherSpec(); err != nil {
		return err
	}
	c.in.seq = info.InSeq

	if err := c.out.changeCipherSpec(); err != nil {
		return err
	}
	c.out.seq = info.OutSeq

	return nil
}

// GetRawConn returns network connection.
func (c *Conn) GetRawConn() net.Conn {
	return c.conn
}

// GetTLSInfo returns TLSInfo
func (c *Conn) GetTLSInfo() *TransferTLSInfo {
	if c.info == nil {
		return nil
	}

	c.info.InSeq = c.in.seq
	c.info.OutSeq = c.out.seq
	if c.rawInput.Len() != 0 {
		tmpBuf := bytes.NewBuffer(make([]byte, c.rawInput.Len()))
		io.Copy(tmpBuf, &c.rawInput)
		c.info.RawInput = tmpBuf.Next(tmpBuf.Len())
	}

	if c.input.Len() != 0 {
		tmpBuf := bytes.NewBuffer(make([]byte, c.input.Len()))
		io.Copy(tmpBuf, &c.input)
		c.info.Input = tmpBuf.Next(tmpBuf.Len())
	}

	return c.info
}

// GetConnectionState records basic TLS details about the connection.
func (c *Conn) GetConnectionState() gotls.ConnectionState {
	c.handshakeMutex.Lock()
	defer c.handshakeMutex.Unlock()

	var state gotls.ConnectionState
	if c.handshakeComplete() {
		state.HandshakeComplete = true
	} else {
		state.HandshakeComplete = false
	}

	state.ServerName = c.serverName

	if c.handshakeComplete() {
		state.Version = c.vers
		state.NegotiatedProtocol = c.clientProtocol
		state.DidResume = c.didResume
		state.NegotiatedProtocolIsMutual = !c.clientProtocolFallback
		state.CipherSuite = c.cipherSuite
		state.PeerCertificates = c.peerCertificates
		state.VerifiedChains = c.verifiedChains
		state.SignedCertificateTimestamps = c.scts
		state.OCSPResponse = c.ocspResponse
		if !c.didResume {
			if c.clientFinishedIsFirst {
				state.TLSUnique = c.clientFinished[:]
			} else {
				state.TLSUnique = c.serverFinished[:]
			}
		}
	}

	return state
}

// GetRawConn returns network connection.
func (c *Conn) SetALPN(alpn string) {
	haveNPN := false
	for _, p := range c.config.NextProtos {
		if p == alpn {
			haveNPN = true
			break
		}
	}
	if !haveNPN {
		c.config.NextProtos = append(c.config.NextProtos, alpn)
	}
}
