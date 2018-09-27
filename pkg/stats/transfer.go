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

package stats

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"net"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/types"
	metrics "github.com/rcrowley/go-metrics"
)

// TransferData keeps information for go-metrics
type TransferData struct {
	MetricsType   string
	MetricsKey    string
	MetricsValues []int64
}

const (
	metricsCounter   = "counter"
	metricsGauge     = "guage"
	metricsHistogram = "histogram"
)

func init() {
	gob.Register(new(TransferData))
}

// makesTransferData get all registered metrics data as a map[string]map[string][]TransferData
// the map will be gob encoded to transfer
func makesTransferData() ([]byte, error) {
	transfers := make(map[string]map[string][]TransferData)
	// Locks all data
	reg.mutex.RLock()
	for key, r := range reg.registries {
		m := make(map[string][]TransferData)
		r.Each(func(key string, i interface{}) {
			values := strings.SplitN(key, sep, 3)
			if len(values) != 3 { // unexepcted metrics, ignore
				return
			}
			namespace := values[1]
			metricsKey := values[2]
			data := TransferData{
				MetricsKey: metricsKey,
			}
			switch metric := i.(type) {
			case metrics.Counter:
				data.MetricsType = metricsCounter
				data.MetricsValues = []int64{metric.Count()}
			case metrics.Gauge:
				data.MetricsType = metricsGauge
				data.MetricsValues = []int64{metric.Value()}
			case metrics.Histogram:
				h := metric.Snapshot()
				data.MetricsType = metricsHistogram
				data.MetricsValues = h.Sample().Values()
			default: //unsupport metrics, ignore
				return
			}
			m[namespace] = append(m[namespace], data)
		})
		transfers[key] = m
	}
	reg.mutex.RUnlock()
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(transfers); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// readTransferData gets the gob encoded data, makes it as go-metrics data
func readTransferData(b []byte) error {
	transfers := make(map[string]map[string][]TransferData)
	buf := bytes.NewBuffer(b)
	if err := gob.NewDecoder(buf).Decode(&transfers); err != nil {
		return err
	}
	for typ, datas := range transfers {
		for namespace, list := range datas {
			for _, d := range list {
				s := NewStats(typ, namespace)
				switch d.MetricsType {
				case metricsCounter:
					s.Counter(d.MetricsKey).Inc(d.MetricsValues[0])
				case metricsGauge:
					s.Gauge(d.MetricsKey).Update(d.MetricsValues[0])
				case metricsHistogram:
					h := s.Histogram(d.MetricsKey)
					for _, v := range d.MetricsValues {
						h.Update(v)
					}
				}
			}
		}
	}
	return nil
}

// TransferDomainSocket represents unix socket for listener
var TransferDomainSocket = path.Join(filepath.Dir(os.Args[0]), "stats.sock")

// TransferServer starts a unix socket, lasts 10 seconds and 2*$gracefultime}, receive metrics datas
// When serves a conn, sends a message to chan
func TransferServer(gracefultime time.Duration, ch chan<- bool) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("transfer metrics server panic %v", r)
		}
	}()
	if os.Getenv(types.GracefulRestart) != "true" {
		return
	}
	if _, err := os.Stat(TransferDomainSocket); err == nil {
		os.Remove(TransferDomainSocket)
	}
	ln, err := net.Listen("unix", TransferDomainSocket)
	if err != nil {
		log.DefaultLogger.Errorf("transfer metrics net listen error %v", err)
		return
	}
	defer ln.Close()
	log.DefaultLogger.Infof("transfer metrics server start")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.DefaultLogger.Errorf("transfer metrics server panic %v", r)
			}
		}()
		for {
			conn, err := ln.Accept()
			if err != nil {
				if ope, ok := err.(*net.OpError); ok && (ope.Op == "accept") {
					log.DefaultLogger.Infof("transfer metrics server listener closed")
				} else {
					log.DefaultLogger.Errorf("transfer metrics server accept error %v", err)
				}
				return
			}
			log.DefaultLogger.Infof("transfer metrics accept")
			go func() {
				serveConn(conn)
				if ch != nil {
					select {
					case ch <- true:
					case <-time.After(10 * time.Second): // write timeout
					}
				}
			}()
		}
	}()
	select {
	case <-time.After(gracefultime*2 + time.Second*10):
		log.DefaultLogger.Infof("transfer metrics server exit")
	}
}

// TransferMetrics sends metrics data to unix socket
// If wait is true, will wait server response, with ${timeout}
// If wait is false, timeout is useless
func TransferMetrics(wait bool, timeout time.Duration) {
	body, err := makesTransferData()
	if err != nil {
		log.DefaultLogger.Errorf("transfer metrics get metrics data error: %v", err)
		return
	}
	transferMetrics(body, wait, timeout)
}

func transferMetrics(body []byte, wait bool, timeout time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			log.DefaultLogger.Errorf("transfer metrics send data error: %v", r)
		}
	}()
	conn, err := net.Dial("unix", TransferDomainSocket)
	if err != nil {
		log.DefaultLogger.Errorf("transfer metrics dial unix socket failed:%v", err)
		return
	}
	defer conn.Close()
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(body)))
	if _, err := conn.Write(header); err != nil {
		log.DefaultLogger.Errorf("transfer metrics send header error: %v", err)
		return
	}
	if _, err := conn.Write(body); err != nil {
		log.DefaultLogger.Errorf("transfer metrics send body error: %v", err)
	}
	if wait {
		conn.SetReadDeadline(time.Now().Add(timeout))
		resp := make([]byte, 1)
		_, err := conn.Read(resp)
		if err != nil {
			log.DefaultLogger.Errorf("transfer metrics get response error: %v", err)
		}
		log.DefaultLogger.Infof("transfer metrics get reponse status: %v", resp[0])
	}
}

/**
*  transfer protocol
*  request:
*  	header: data length (4 bytes, uint32, bigendian)
*  	body: data (data length bytes)
*  response:
* 	header: status code (1 bytes, 0 means ok, 1 means failed)
**/
func read(conn net.Conn, size int) ([]byte, error) {
	if size == 0 {
		return nil, nil
	}
	b := make([]byte, size)
	var n, off int
	var err error
	for {
		n, err = conn.Read(b[off:])
		if err != nil {
			return nil, err
		}
		off += n
		if off == size {
			return b, nil
		}
	}
}
func readHeader(conn net.Conn) (int, error) {
	b, err := read(conn, 4)
	if err != nil {
		return 0, err
	}
	return int(binary.BigEndian.Uint32(b)), nil
}

func serveConn(conn net.Conn) {
	b := make([]byte, 1)
	if err := handler(conn); err != nil {
		b[0] = 0x01
	}
	conn.Write(b)
}

func handler(conn net.Conn) error {
	size, err := readHeader(conn)
	if err != nil {
		log.DefaultLogger.Errorf("transfer metrics read header error: %v", err)
		return err
	}
	body, err := read(conn, size)
	if err != nil {
		log.DefaultLogger.Errorf("transfer metrics read body error: %v", err)
		return err
	}
	if err := readTransferData(body); err != nil {
		log.DefaultLogger.Errorf("transfer metrics parse body error: %v", err)
		return err
	}
	return nil
}
