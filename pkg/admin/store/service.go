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

package store

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"runtime/debug"

	"sync"

	"github.com/alipay/sofa-mosn/pkg/log"
	"github.com/alipay/sofa-mosn/pkg/metrics"
)

var lock = new(sync.Mutex)

type service struct {
	start bool
	*http.Server
	name string
	init func()
	exit func()
}

var services []*service
var listeners []net.Listener

func ListServiceListenersFile() ([]*os.File, error) {
	if len(listeners) == 0 {
		return nil, nil
	}

	files := make([]*os.File, len(listeners))

	for i, l := range listeners {
		var ok bool
		var tl *net.TCPListener
		if tl, ok = l.(*net.TCPListener); !ok {
			return nil, errors.New("listener type is error")
		}
		file, err := tl.File()
		if err != nil {
			log.DefaultLogger.Errorf("fail to get listener %s file descriptor: %v", tl.Addr().String(), err)
			return nil, errors.New("fail to get listener fd") //stop reconfigure
		}
		files[i] = file
	}
	return files, nil
}

func AddService(s *http.Server, name string, init func(), exit func()) {
	lock.Lock()
	defer lock.Unlock()
	for i, srv := range services {
		if srv.Addr == s.Addr {
			services[i] = &service{false, s, name, init, exit}
			return
		}
	}
	services = append(services, &service{false, s, name, init, exit})
}

func StartService(inheritListeners []net.Listener) error {
	for _, srv := range services {
		if srv.start {
			continue
		}
		var err error
		var ln net.Listener
		var saddr *net.TCPAddr

		s := srv
		saddr, err = net.ResolveTCPAddr("tcp", s.Addr)
		if err != nil {
			log.StartLogger.Fatalln("[inheritListener] not valid:", s.Addr)
		}

		for i, l := range inheritListeners {
			if l == nil {
				continue
			}
			addr, err := net.ResolveTCPAddr("tcp", l.Addr().String())
			if err != nil {
				log.StartLogger.Fatalln("[inheritListener] not valid: ", l.Addr().String())
			}

			if addr.Port == saddr.Port {
				ln = l
				inheritListeners[i] = nil
				log.StartLogger.Infof("inherit listener addr: %s", ln.Addr().String())
				break
			}
		}

		if ln == nil {
			ln, err = net.Listen("tcp", s.Addr)
			if err != nil {
				return err
			}
		}
		listeners = append(listeners, ln)
		if s.name != "" {
			log.StartLogger.Infof("start service %s on %s", s.name, ln.Addr().String())
		}
		if s.init != nil {
			s.init()
		}
		s.start = true

		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.DefaultLogger.Errorf("service %s panic %v", s.name, r)
				}
			}()

			// set metrics
			metrics.AddListenerAddr(s.Addr)

			s.Serve(ln)
		}()
	}
	return nil
}

func StopService() {
	for _, srv := range services {
		s := srv
		if s.exit != nil {
			s.exit()
		}
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.DefaultLogger.Errorf("panic %v\n%s", r, string(debug.Stack()))
				}
			}()

			s.Shutdown(context.Background())
		}()
	}
	services = services[:0]
	listeners = listeners[:0]
}
