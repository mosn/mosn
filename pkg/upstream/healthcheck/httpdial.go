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

package healthcheck

import (
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

const (
	HTTPCheckConfigKey = "httpCheckConfig"
)

const (
	defaultTimeout = 30
)

func init() {
	httpDialSessionFactory := &HTTPDialSessionFactory{}
	RegisterSessionFactory(protocol.HTTP1, httpDialSessionFactory)
}

type HttpCheckConfig struct {
	Port    int    `json:"port,omitempty"`
	Timeout int    `json:"timeout,omitempty"`
	Path    string `json:"path,omitempty"`
}

type HTTPDialSession struct {
	client   *http.Client
	timeout  int
	checkUrl string
	*url.URL
}

type HTTPDialSessionFactory struct{}

func (f *HTTPDialSessionFactory) NewSession(cfg map[string]interface{}, host types.Host) types.HealthCheckSession {
	httpDail := &HTTPDialSession{}

	v, ok := cfg[HTTPCheckConfigKey]
	if !ok {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] httpCheckConfig is not config fallback to tcpDial")
		tcpDialSessionFactory := &TCPDialSessionFactory{}
		return tcpDialSessionFactory.NewSession(cfg, host)
	}

	httpCheckConfig, ok := v.(*HttpCheckConfig)
	if !ok {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] httpCheckConfig covert error %+v %+v", reflect.TypeOf(v), v)
		return nil
	}
	if httpCheckConfig.Port <= 0 || httpCheckConfig.Port > 65535 {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] httpCheckConfig port config error %+v", httpCheckConfig)
		return nil
	}

	uri := &url.URL{}
	uri.Scheme = "http"
	hostIp := strings.Split(host.AddressString(), ":")
	// re-config http check port
	uri.Host = hostIp[0] + ":" + strconv.Itoa(httpCheckConfig.Port)
	uri.Path = httpCheckConfig.Path

	if httpCheckConfig.Timeout > 0 {
		httpDail.timeout = httpCheckConfig.Timeout
	} else {
		httpDail.timeout = defaultTimeout
	}

	httpDail.URL = uri
	httpDail.client = &http.Client{
		Timeout: time.Second * time.Duration(httpDail.timeout),
	}
	httpDail.checkUrl = httpDail.String()
	return httpDail
}

func (s *HTTPDialSession) CheckHealth() bool {
	// default dial timeout, maybe already timeout by checker
	resp, err := s.client.Get(s.checkUrl)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] http check for host %s error: %v", s.String(), err)
		return false
	}
	defer resp.Body.Close()

	result := resp.StatusCode == http.StatusOK
	if !result {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] http check for host %s failed, statuscode: %+v", s.String(), resp.StatusCode)
	} else {
		if log.DefaultLogger.GetLogLevel() > log.DEBUG {
			log.DefaultLogger.Debugf("[upstream] [health check] [httpdial session] http check for host %s succeed", s.String())
		}
	}

	return result
}

func (s *HTTPDialSession) OnTimeout() {
	log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] http check for host %s timeout", s.checkUrl)
}
