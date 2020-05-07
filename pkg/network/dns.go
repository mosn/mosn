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
	"strings"
	"time"

	"github.com/miekg/dns"
	v2 "mosn.io/mosn/pkg/config/v2"
	"mosn.io/mosn/pkg/log"
)

const (
	V4Only uint8 = iota
	V6Only
	Auto
)

var DefaultResolverFile string = "/etc/resolv.conf"

type DnsResolver struct {
	clientConfig *dns.ClientConfig
	client       *dns.Client
}

type DnsResponse struct {
	Address string
	Ttl     time.Duration
}

func NewDnsResolver(config *v2.DnsResolverConfig) *DnsResolver {
	clientConfig := &dns.ClientConfig{
		Servers:  config.Servers,
		Search:   config.Search,
		Port:     config.Port,
		Ndots:    config.Ndots,
		Timeout:  config.Timeout,
		Attempts: config.Attempts,
	}
	if clientConfig.Port == "" {
		clientConfig.Port = "53"
	}
	return newDnsResolver(clientConfig)
}

func NewDnsResolverFromFile(configFile string, resolverPort string) *DnsResolver {
	var err error
	var config *dns.ClientConfig
	if configFile == "" {
		configFile = DefaultResolverFile
	}
	if resolverPort == "" {
		resolverPort = "53"
	}

	config, err = dns.ClientConfigFromFile(configFile)
	if err != nil {
		if log.DefaultLogger.GetLogLevel() >= log.ERROR {
			log.DefaultLogger.Errorf("[network] [dns] read config from resolve file failed: %s", err)
		}
		return nil
	}
	config.Port = resolverPort

	return newDnsResolver(config)
}

func newDnsResolver(config *dns.ClientConfig) *DnsResolver {
	return &DnsResolver{
		clientConfig: config,
		client: &dns.Client{
			Net:            "",
			UDPSize:        0,
			TLSConfig:      nil,
			Dialer:         nil,
			Timeout:        time.Duration(config.Timeout) * time.Second,
			DialTimeout:    0,
			ReadTimeout:    0,
			WriteTimeout:   0,
			TsigSecret:     nil,
			SingleInflight: false,
		},
	}
}

func getDnsType(dnsType v2.DnsLookupFamily) uint16 {
	switch dnsType {
	case v2.V4Only:
		return dns.TypeA
	case v2.V6Only:
		return dns.TypeAAAA
	}
	return dns.TypeA
}

func (dr *DnsResolver) DnsResolve(dnsAddr string, dnsLookupFamily v2.DnsLookupFamily) *[]DnsResponse {
	dnsQueryType := getDnsType(dnsLookupFamily)
	msg := new(dns.Msg)
	addrs := dr.clientConfig.NameList(dnsAddr)
	var dnsRsp []DnsResponse
	for _, server := range dr.clientConfig.Servers {
		//try from first server by default
		for _, addr := range addrs {
			msg.SetQuestion(addr, dnsQueryType)
			r, _, err := dr.client.Exchange(msg, server+":"+dr.clientConfig.Port)

			if err != nil {
				if log.DefaultLogger.GetLogLevel() >= log.INFO{
					log.DefaultLogger.Infof("[network] [dns] resolve addr: %s failed, server: %s, err: %s", addr, server, err)
				}
				if strings.Contains(err.Error(), "i/o timeout") {
					break
				}
				continue
			}
			if r.Rcode != dns.RcodeSuccess {
				continue
			}
			switch dnsQueryType {
			case dns.TypeA:
				for _, ans := range r.Answer {
					if a, ok := ans.(*dns.A); ok {
						dnsRsp = append(dnsRsp, DnsResponse{
							Address: a.A.String(),
							Ttl:     time.Duration(ans.Header().Ttl) * time.Second,
						})
					}
				}
			case dns.TypeAAAA:
				for _, ans := range r.Answer {
					if a, ok := ans.(*dns.AAAA); ok {
						dnsRsp = append(dnsRsp, DnsResponse{
							Address: a.AAAA.String(),
							Ttl:     time.Duration(ans.Header().Ttl) * time.Second,
						})
					}
				}
			}
			if len(dnsRsp) > 0 {
				return &dnsRsp
			}
		}
	}

	if log.DefaultLogger.GetLogLevel() >= log.ERROR {
		log.DefaultLogger.Errorf("[network] [dns] resolve addr: %s failed.", dnsAddr)
	}

	return nil
}
