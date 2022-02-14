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
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/protocol"
	"mosn.io/mosn/pkg/types"
)

const (
	HTTPCheckConfigKey = "http_check_config"
)

var defaultTimeout = api.DurationConfig{time.Second * 30}

func init() {
	httpDialSessionFactory := &HTTPDialSessionFactory{}
	RegisterSessionFactory(protocol.HTTP1, httpDialSessionFactory)
}

type HttpCheckConfig struct {
	Port    int                `json:"port,omitempty"`
	Timeout api.DurationConfig `json:"timeout,omitempty"`
	Path    string             `json:"path,omitempty"`
}

type HTTPDialSession struct {
	client   *http.Client
	timeout  time.Duration
	checkUrl string
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
		httpCheckConfigBytes, err := json.Marshal(v)
		if err != nil {
			log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] httpCheckConfig covert %+v error %+v %+v", reflect.TypeOf(v), v, err)
			return nil
		}
		httpCheckConfig = &HttpCheckConfig{}
		if err := json.Unmarshal(httpCheckConfigBytes, httpCheckConfig); err != nil {
			log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] httpCheckConfig Unmarshal %+v error %+v %+v", reflect.TypeOf(v), v, err)
			return nil
		}
	}

	uri := &url.URL{}
	uri.Scheme = "http"

	hostIp, _, err := net.SplitHostPort(host.AddressString())
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] host=%s parse error %+v", host.AddressString(), err)
		return nil
	}

	if httpCheckConfig.Port > 0 && httpCheckConfig.Port < 65535 {
		// re-config http check port
		uri.Host = hostIp + ":" + strconv.Itoa(httpCheckConfig.Port)
		uri.Path = httpCheckConfig.Path
	} else {
		// use rpc port as http check port
		log.DefaultLogger.Warnf("[upstream] [health check] [httpdial session] httpCheckConfig port config error %+v", httpCheckConfig)
		uri.Host = host.AddressString()
		uri.Path = httpCheckConfig.Path
	}

	if httpCheckConfig.Timeout.Duration > 0 {
		httpDail.timeout = httpCheckConfig.Timeout.Duration
	} else {
		httpDail.timeout = defaultTimeout.Duration
	}

	httpDail.client = &http.Client{
		Timeout: httpDail.timeout,
	}
	httpDail.checkUrl = uri.String()
	log.DefaultLogger.Infof("[upstream] [health check] [httpdial session]  create a health check success for %s", httpDail.checkUrl)
	return httpDail
}

func (s *HTTPDialSession) CheckHealth() bool {
	// default dial timeout, maybe already timeout by checker
	resp, err := s.client.Get(s.checkUrl)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] http check for host %s error: %v", s.checkUrl, err)
		return false
	}
	defer resp.Body.Close()

	result := resp.StatusCode == http.StatusOK
	if !result {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] http check for host %s failed, statuscode: %+v", s.checkUrl, resp.StatusCode)
	} else {
		if log.DefaultLogger.GetLogLevel() > log.DEBUG {
			log.DefaultLogger.Debugf("[upstream] [health check] [httpdial session] http check for host %s succeed", s.checkUrl)
		}
	}

	return result
}

func (s *HTTPDialSession) OnTimeout() {
	log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] http check for host %s timeout", s.checkUrl)
}
