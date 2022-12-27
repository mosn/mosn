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
	"strings"
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

type CodeRange struct {
	Start int
	End   int
}

type HttpCheckConfig struct {
	Port    int                `json:"port,omitempty"`
	Timeout api.DurationConfig `json:"timeout,omitempty"`
	Path    string             `json:"path,omitempty"`
	Method  string             `json:"method,omitempty"`
	Scheme  string             `json:"scheme,omitempty"`
	Domain  string             `json:"domain,omitempty"`
	Codes   []CodeRange        `json:"codes,omitempty"`
}

type HTTPDialSession struct {
	client  *http.Client
	timeout time.Duration
	request *http.Request
	Codes   []CodeRange
}

type HTTPDialSessionFactory struct{}

func (f *HTTPDialSessionFactory) NewSession(cfg map[string]interface{}, host types.Host) types.HealthCheckSession {
	httpDial := &HTTPDialSession{}

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
	if httpCheckConfig.Scheme == "" {
		uri.Scheme = "http"
	} else {
		uri.Scheme = httpCheckConfig.Scheme
	}

	hostIp, _, err := net.SplitHostPort(host.AddressString())
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] host=%s parse error %+v", host.AddressString(), err)
		return nil
	}

	if httpCheckConfig.Port > 0 && httpCheckConfig.Port < 65535 {
		// re-config http check port
		uri.Host = hostIp + ":" + strconv.Itoa(httpCheckConfig.Port)
	} else {
		// use rpc port as http check port
		log.DefaultLogger.Warnf("[upstream] [health check] [httpdial session] httpCheckConfig port config error %+v", httpCheckConfig)
		uri.Host = host.AddressString()
	}

	ind := strings.IndexByte(httpCheckConfig.Path, '?')
	if ind < 0 {
		uri.Path = httpCheckConfig.Path
	} else {
		uri.Path = httpCheckConfig.Path[:ind]
		if len(httpCheckConfig.Path) > ind+1 {
			uri.RawQuery = httpCheckConfig.Path[ind+1:]
		}
	}

	if httpCheckConfig.Timeout.Duration > 0 {
		httpDial.timeout = httpCheckConfig.Timeout.Duration
	} else {
		httpDial.timeout = defaultTimeout.Duration
	}

	httpDial.client = &http.Client{
		Timeout: httpDial.timeout,
	}

	httpDial.request, err = http.NewRequest(httpCheckConfig.Method, uri.String(), nil)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session]  create health check request failed, %v", err)
		return nil
	}

	if httpCheckConfig.Domain != "" {
		httpDial.request.Host = httpCheckConfig.Domain
	}

	httpDial.Codes = httpCheckConfig.Codes

	log.DefaultLogger.Infof("[upstream] [health check] [httpdial session]  create a health check success for %s", uri.String())
	return httpDial
}

func (s *HTTPDialSession) verifyCode(code int) bool {
	// default: [200, 200]
	if len(s.Codes) == 0 {
		return code == 200
	}
	for _, codeRange := range s.Codes {
		if code >= codeRange.Start && code <= codeRange.End {
			return true
		}
	}
	return false
}

func (s *HTTPDialSession) CheckHealth() bool {
	// default dial timeout, maybe already timeout by checker
	resp, err := s.client.Do(s.request)
	if err != nil {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] http check for host %s error: %v", s.request.URL.String(), err)
		return false
	}
	defer resp.Body.Close()

	result := s.verifyCode(resp.StatusCode)
	if !result {
		log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] http check for host %s failed, statuscode: %+v", s.request.URL.String(), resp.StatusCode)
	} else {
		if log.DefaultLogger.GetLogLevel() > log.DEBUG {
			log.DefaultLogger.Debugf("[upstream] [health check] [httpdial session] http check for host %s succeed", s.request.URL.String())
		}
	}

	return result
}

func (s *HTTPDialSession) OnTimeout() {
	log.DefaultLogger.Errorf("[upstream] [health check] [httpdial session] http check for host %s timeout", s.request.URL.String())
}
