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

package config

import (
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/alipay/sofa-mosn/pkg/api/v2"
)

func TestParseClusterHealthCheckConf(t *testing.T) {
	healthCheckConfigStr := `{
		  "protocol": "SofaRpc",
		  "timeout": "90s",
		  "healthy_threshold": 2,
		  "unhealthy_threshold": 2,
		  "interval": "5s",
		  "interval_jitter": 0,
		  "check_path": ""
		}`

	var ccc ClusterHealthCheckConfig

	json.Unmarshal([]byte(healthCheckConfigStr), &ccc)
	want := v2.HealthCheck{
		Protocol:           "SofaRpc",
		Timeout:            90 * time.Second,
		HealthyThreshold:   2,
		UnhealthyThreshold: 2,
		Interval:           5 * time.Second,
		IntervalJitter:     0,
		CheckPath:          "",
		ServiceName:        "",
	}

	type args struct {
		c *ClusterHealthCheckConfig
	}
	tests := []struct {
		name string
		args args
		want v2.HealthCheck
	}{
		// TODO: Add test cases.
		{
			name: "test1",
			args: args{
				c: &ccc,
			},
			want: want,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseClusterHealthCheckConf(tt.args.c); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseClusterHealthCheckConf() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFilterChainJSONFile(t *testing.T) {
	var filterchan FilterChain
	filterchanStr := `{
             "match":"test",
             "tls_context":{
               "status": true,
               "inspector": true,
               "server_name": "hello.com",
               "ca_cert": "-----BEGIN CERTIFICATE-----\nMIIDMjCCAhoCCQDaFC8PcSS5qTANBgkqhkiG9w0BAQsFADBbMQswCQYDVQQGEwJD\nTjEKMAgGA1UECAwBYTEKMAgGA1UEBwwBYTEKMAgGA1UECgwBYTEKMAgGA1UECwwB\nYTEKMAgGA1UEAwwBYTEQMA4GCSqGSIb3DQEJARYBYTAeFw0xODA2MTQwMjQyMjVa\nFw0xOTA2MTQwMjQyMjVaMFsxCzAJBgNVBAYTAkNOMQowCAYDVQQIDAFhMQowCAYD\nVQQHDAFhMQowCAYDVQQKDAFhMQowCAYDVQQLDAFhMQowCAYDVQQDDAFhMRAwDgYJ\nKoZIhvcNAQkBFgFhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArbNc\nmOXvvOqZgJdxMIiklE8lykVvz5d7QZ0+LDVG8phshq9/woigfB1aFBAI36/S5LZQ\n5Fd0znblSa+LOY06jdHTkbIBYFlxH4tdRaD0B7DbFzR5bpzLv2Q+Zf5u5RI73Nky\nH8CjW9QJjboArHkwm0YNeENaoR/96nYillgYLnunol4h0pxY7ZC6PpaB1EBaTXcz\n0iIUX4ktUJQmYZ/DFzB0oQl9IWOj18ml2wYzu9rYsySzj7EPnDOOebsRfS5hl3fz\nHi4TC4PDh0mQwHqDQ4ncztkybuRSXFQ6RzEPdR5qtp9NN/G/TlfyB0CET3AFmGkp\nE2irGoF/JoZXEDeXmQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQApzhQLS7fAcExZ\nx1S+hcy7lLF8QcPlsiH32SnLFg5LPy4prz71mebUchmt97t4T3tSWzwXi8job7Q2\nONYc6sr1LvaFtg7qoCfz5fPP5x+kKDkEPwCDJSTVPcXP+UtA407pxX8KPRN8Roay\ne3oGcmNqVu/DkkufkIL3PBg41JEMovWtKD+PXmeBafc4vGCHSJHJBmzMe5QtwHA0\nss/A9LHPaq3aLcIyFr8x7clxc7zZVaim+lVfNV3oPBnB4gU7kLFVT0zOhkM+V1A4\nQ5GVbGAu4R7ItY8kJ2b7slre0ajPUp2FMregt4mnUM3mu1nbltVhtoknXqHHMGgN\n4Lh4JfNx\n-----END CERTIFICATE-----\n",
               "cert_chain": "-----BEGIN CERTIFICATE-----\nMIIDJTCCAg0CAQEwDQYJKoZIhvcNAQELBQAwWzELMAkGA1UEBhMCQ04xCjAIBgNV\nBAgMAWExCjAIBgNVBAcMAWExCjAIBgNVBAoMAWExCjAIBgNVBAsMAWExCjAIBgNV\nBAMMAWExEDAOBgkqhkiG9w0BCQEWAWEwHhcNMTgwNjE0MDMxMzQyWhcNMTkwNjE0\nMDMxMzQyWjBWMQswCQYDVQQGEwJDTjEKMAgGA1UECAwBYTEKMAgGA1UECgwBYTEK\nMAgGA1UECwwBYTERMA8GA1UEAwwIdGVzdC5jb20xEDAOBgkqhkiG9w0BCQEWAWEw\nggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCrPq+Mo0nS3dJU1qGFwlIB\ni9HqRm5RGcfps+0W5LjEhqUKxKUweRrwDaIxpiSqjKeehz9DtLUpXBD29pHuxODU\nVsMH2U1AGWn9l4jMnP6G5iTMPJ3ZTXszeqALe8lm/f807ZA0C7moc+t7/d3+b6d2\nlnwR+yWbIZJUu2qw+HrR0qPpNlBP3EMtlQBOqf4kCl6TfpqrGfc9lW0JjuE6Taq3\ngSIIgzCsoUFe30Yemho/Pp4zA9US97DZjScQLQAGiTsCRDBASxXGzODQOfZL3bCs\n2w//1lqGjmhp+tU1nR4MRN+euyNX42ioEz111iB8y0VzuTIsFBWwRTKK1SF7YSEb\nAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABnRM9JJ21ZaujOTunONyVLHtmxUmrdr\n74OJW8xlXYEMFu57Wi40+4UoeEIUXHviBnONEfcITJITYUdqve2JjQsH2Qw3iBUr\nmsFrWS25t/Krk2FS2cKg8B9azW2+p1mBNm/FneMv2DMWHReGW0cBp3YncWD7OwQL\n9NcYfXfgBgHdhykctEQ97SgLHDKUCU8cPJv14eZ+ehIPiv8cDWw0mMdMeVK9q71Y\nWn2EgP7HzVgdbj17nP9JJjNvets39gD8bU0g2Lw3wuyb/j7CHPBBzqxh+a8pihI5\n3dRRchuVeMQkMuukyR+/A8UrBMA/gCTkXIcP6jKO1SkKq5ZwlMmapPc=\n-----END CERTIFICATE-----\n",
               "private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAqz6vjKNJ0t3SVNahhcJSAYvR6kZuURnH6bPtFuS4xIalCsSl\nMHka8A2iMaYkqoynnoc/Q7S1KVwQ9vaR7sTg1FbDB9lNQBlp/ZeIzJz+huYkzDyd\n2U17M3qgC3vJZv3/NO2QNAu5qHPre/3d/m+ndpZ8EfslmyGSVLtqsPh60dKj6TZQ\nT9xDLZUATqn+JApek36aqxn3PZVtCY7hOk2qt4EiCIMwrKFBXt9GHpoaPz6eMwPV\nEvew2Y0nEC0ABok7AkQwQEsVxszg0Dn2S92wrNsP/9Zaho5oafrVNZ0eDETfnrsj\nV+NoqBM9ddYgfMtFc7kyLBQVsEUyitUhe2EhGwIDAQABAoIBAG2Bj5ca0Fmk+hzA\nh9fWdMSCWgE7es4n81wyb/nE15btF1t0dsIxn5VE0qR3P1lEyueoSz+LrpG9Syfy\nc03B3phKxzscrbbAybOeFJ/sASPYxk1IshRE5PT9hJzzUs6mvG1nQWDW4qmjP0Iy\nDKTpV6iRANQqy1iRtlay5r42l6vWwHfRfwAv4ExSS+RgkYcavqOp3e9If2JnFJuo\n7Zds2i7Ux8dURX7lHqKxTt6phgoMmMpvO3lFYVGos7F13OR9NKElzjiefAQbweAt\nt8R+6A1rlIwnfywxET9ZXglfOFK6Q0nqCJhcEcKzT/Xfkd+h9XPACjOObJh3a2+o\nwg9GBFECgYEA2a6JYuFanKzvajFPbSeN1csfI9jPpK2+tB5+BB72dE74B4rjygiG\n0Rb26UjovkYfJJqKuKr4zDL5ziSlJk199Ae2f6T7t7zmyhMlWQtVT12iTQvBINTz\nNerKi5HNVBsCSGj0snbwo8u4QRgTjaIoVqTlOlUQuGqYuZ75l8g35IkCgYEAyWOM\nKagzpGmHWq/0ThN4kkwWOdujxuqrPf4un2WXsir+L90UV7X9wY4mO19pe5Ga2Upu\nXFDsxAZsanf8SbzkTGHvzUobFL7eqsiwaUSGB/cGEtkIyVlAdyW9DhiZFt3i9mEF\nbBsHnEDHPHL4tu+BB8G3WahHjWOnbWZ3NTtP94MCgYEAi3XRmSLtjYER5cPvsevs\nZ7M5oRqvdT7G9divPW6k0MEjEJn/9BjgXqbKy4ylZ/m+zBGinEsVGKXz+wjpMY/m\nCOjEGCUYC5AfgAkiHVkwb6d6asgEFEe6BaoF18MyfBbNsJxlYMzowNeslS+an1vr\nYg9EuMl06+GHNSzPlVl1zZkCgYEAxXx8N2F9eu4NUK4ZafMIGpbIeOZdHbSERp+b\nAq5yasJkT3WB/F04QXVvImv3Gbj4W7r0rEyjUbtm16Vf3sOAMTMdIHhaRCbEXj+9\nVw1eTjM8XoE8b465e92jHk6a2WSvq6IK2i9LcDvJ5QptwZ7uLjgV37L4r7sYtVx0\n69uFGJcCgYEAot7im+Yi7nNsh1dJsDI48liKDjC6rbZoCeF7Tslp8Lt+4J9CA9Ux\nSHyKjg9ujbjkzCWrPU9hkugOidDOmu7tJAxB5cS00qJLRqB5lcPxjOWcBXCdpBkO\n0tdT/xRY/MYLf3wbT95enaPlhfeqBBXKNQDya6nISbfwbMLfNxdZPJ8=\n-----END RSA PRIVATE KEY-----\n",
               "verify_client": true,
               "cipher_suites": "ECDHE-RSA-AES256-GCM-SHA384",
               "ecdh_curves": "P256"
               },
              "filters": [
                {
                  "type": "proxy",
                  "config": {
                    "downstream_protocol": "SofaRpc",
                    "name": "proxy_config",
                    "support_dynamic_route": true,
                    "upstream_protocol": "SofaRpc",
                    "virtual_hosts": [
                      {
                        "name": "sofa",
                        "require_tls": "no",
                        "domains":[
                          "*testwilccard"
                        ],
                        "routers": [
                          {
                            "match": {
                              "headers": [
                                {
                                  "name": "service",
                                  "value": "com.alipay.rpc.common.service.facade.pb.SampleServicePb:1.0",
                                  "regex":false
                                }
                              ]
                            },
                            "route": {
                              "cluster_name": "test_cpp",
                              "cluster_header": {
                                "filter_metadata": {
                                  "mosn.lb": {
                                    "version":"1.1",
                                    "stage":"pre-release",
                                    "label": "gray"
                                  }
                                }
                              }
                            }
                          }
                        ]
                      }
                    ]
                  }
                }
              ]
            }`
	json.Unmarshal([]byte(filterchanStr), &filterchan)
	if filterchan.FilterChainMatch != "test" || len(filterchan.Filters) != 1 ||
		filterchan.Filters[0].Type != "proxy" || filterchan.TLS.ServerName != "hello.com" {
		t.Errorf("TestParseFilterChain Failure")
	}
}

func TestParseProxyFilterJSONFile(t *testing.T) {
	var proxy Proxy
	filterchanStr := `{
                    "downstream_protocol": "SofaRpc",
                    "name": "proxy_config",
                    "support_dynamic_route": true,
                    "upstream_protocol": "SofaRpc",
                    "virtual_hosts": [
                      {
                        "name": "sofa",
                        "require_tls": "no",
                        "domains":[
                          "*testwilccard"
                        ],
                        "routers": [
                          {
                            "match": {
                              "headers": [
                                {
                                  "name": "service",
                                  "value": "com.alipay.rpc.common.service.facade.pb.SampleServicePb:1.0",
                                  "regex":false
                                }
                              ]
                            },
                            "route": {
                              "cluster_name": "test_cpp",
                              "metadata_match": {
                                "filter_metadata": {
                                  "mosn.lb": {
                                    "version":"1.1",
                                    "stage":"pre-release",
                                    "label": "gray"
                                  }
                                }
                              }
                            }
                          }
                        ]
                      }
                    ]
                  }`

	json.Unmarshal([]byte(filterchanStr), &proxy)

	if proxy.Name != "proxy_config" || len(proxy.VirtualHosts) != 1 ||
		proxy.VirtualHosts[0].Name != "sofa" {
		t.Errorf("TestParseProxyFilterJSON Failure")
	}
}

func TestParseTlsJsonFile(t *testing.T) {
	tlscon := TLSConfig{}
	test := `{
              "status": true,
               "inspector": true,
               "server_name": "hello.com",
               "ca_cert": "-----BEGIN CERTIFICATE-----\nMIIDMjCCAhoCCQDaFC8PcSS5qTANBgkqhkiG9w0BAQsFADBbMQswCQYDVQQGEwJD\nTjEKMAgGA1UECAwBYTEKMAgGA1UEBwwBYTEKMAgGA1UECgwBYTEKMAgGA1UECwwB\nYTEKMAgGA1UEAwwBYTEQMA4GCSqGSIb3DQEJARYBYTAeFw0xODA2MTQwMjQyMjVa\nFw0xOTA2MTQwMjQyMjVaMFsxCzAJBgNVBAYTAkNOMQowCAYDVQQIDAFhMQowCAYD\nVQQHDAFhMQowCAYDVQQKDAFhMQowCAYDVQQLDAFhMQowCAYDVQQDDAFhMRAwDgYJ\nKoZIhvcNAQkBFgFhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArbNc\nmOXvvOqZgJdxMIiklE8lykVvz5d7QZ0+LDVG8phshq9/woigfB1aFBAI36/S5LZQ\n5Fd0znblSa+LOY06jdHTkbIBYFlxH4tdRaD0B7DbFzR5bpzLv2Q+Zf5u5RI73Nky\nH8CjW9QJjboArHkwm0YNeENaoR/96nYillgYLnunol4h0pxY7ZC6PpaB1EBaTXcz\n0iIUX4ktUJQmYZ/DFzB0oQl9IWOj18ml2wYzu9rYsySzj7EPnDOOebsRfS5hl3fz\nHi4TC4PDh0mQwHqDQ4ncztkybuRSXFQ6RzEPdR5qtp9NN/G/TlfyB0CET3AFmGkp\nE2irGoF/JoZXEDeXmQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQApzhQLS7fAcExZ\nx1S+hcy7lLF8QcPlsiH32SnLFg5LPy4prz71mebUchmt97t4T3tSWzwXi8job7Q2\nONYc6sr1LvaFtg7qoCfz5fPP5x+kKDkEPwCDJSTVPcXP+UtA407pxX8KPRN8Roay\ne3oGcmNqVu/DkkufkIL3PBg41JEMovWtKD+PXmeBafc4vGCHSJHJBmzMe5QtwHA0\nss/A9LHPaq3aLcIyFr8x7clxc7zZVaim+lVfNV3oPBnB4gU7kLFVT0zOhkM+V1A4\nQ5GVbGAu4R7ItY8kJ2b7slre0ajPUp2FMregt4mnUM3mu1nbltVhtoknXqHHMGgN\n4Lh4JfNx\n-----END CERTIFICATE-----\n",
               "cert_chain": "-----BEGIN CERTIFICATE-----\nMIIDJTCCAg0CAQEwDQYJKoZIhvcNAQELBQAwWzELMAkGA1UEBhMCQ04xCjAIBgNV\nBAgMAWExCjAIBgNVBAcMAWExCjAIBgNVBAoMAWExCjAIBgNVBAsMAWExCjAIBgNV\nBAMMAWExEDAOBgkqhkiG9w0BCQEWAWEwHhcNMTgwNjE0MDMxMzQyWhcNMTkwNjE0\nMDMxMzQyWjBWMQswCQYDVQQGEwJDTjEKMAgGA1UECAwBYTEKMAgGA1UECgwBYTEK\nMAgGA1UECwwBYTERMA8GA1UEAwwIdGVzdC5jb20xEDAOBgkqhkiG9w0BCQEWAWEw\nggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCrPq+Mo0nS3dJU1qGFwlIB\ni9HqRm5RGcfps+0W5LjEhqUKxKUweRrwDaIxpiSqjKeehz9DtLUpXBD29pHuxODU\nVsMH2U1AGWn9l4jMnP6G5iTMPJ3ZTXszeqALe8lm/f807ZA0C7moc+t7/d3+b6d2\nlnwR+yWbIZJUu2qw+HrR0qPpNlBP3EMtlQBOqf4kCl6TfpqrGfc9lW0JjuE6Taq3\ngSIIgzCsoUFe30Yemho/Pp4zA9US97DZjScQLQAGiTsCRDBASxXGzODQOfZL3bCs\n2w//1lqGjmhp+tU1nR4MRN+euyNX42ioEz111iB8y0VzuTIsFBWwRTKK1SF7YSEb\nAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABnRM9JJ21ZaujOTunONyVLHtmxUmrdr\n74OJW8xlXYEMFu57Wi40+4UoeEIUXHviBnONEfcITJITYUdqve2JjQsH2Qw3iBUr\nmsFrWS25t/Krk2FS2cKg8B9azW2+p1mBNm/FneMv2DMWHReGW0cBp3YncWD7OwQL\n9NcYfXfgBgHdhykctEQ97SgLHDKUCU8cPJv14eZ+ehIPiv8cDWw0mMdMeVK9q71Y\nWn2EgP7HzVgdbj17nP9JJjNvets39gD8bU0g2Lw3wuyb/j7CHPBBzqxh+a8pihI5\n3dRRchuVeMQkMuukyR+/A8UrBMA/gCTkXIcP6jKO1SkKq5ZwlMmapPc=\n-----END CERTIFICATE-----\n",
               "private_key": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAqz6vjKNJ0t3SVNahhcJSAYvR6kZuURnH6bPtFuS4xIalCsSl\nMHka8A2iMaYkqoynnoc/Q7S1KVwQ9vaR7sTg1FbDB9lNQBlp/ZeIzJz+huYkzDyd\n2U17M3qgC3vJZv3/NO2QNAu5qHPre/3d/m+ndpZ8EfslmyGSVLtqsPh60dKj6TZQ\nT9xDLZUATqn+JApek36aqxn3PZVtCY7hOk2qt4EiCIMwrKFBXt9GHpoaPz6eMwPV\nEvew2Y0nEC0ABok7AkQwQEsVxszg0Dn2S92wrNsP/9Zaho5oafrVNZ0eDETfnrsj\nV+NoqBM9ddYgfMtFc7kyLBQVsEUyitUhe2EhGwIDAQABAoIBAG2Bj5ca0Fmk+hzA\nh9fWdMSCWgE7es4n81wyb/nE15btF1t0dsIxn5VE0qR3P1lEyueoSz+LrpG9Syfy\nc03B3phKxzscrbbAybOeFJ/sASPYxk1IshRE5PT9hJzzUs6mvG1nQWDW4qmjP0Iy\nDKTpV6iRANQqy1iRtlay5r42l6vWwHfRfwAv4ExSS+RgkYcavqOp3e9If2JnFJuo\n7Zds2i7Ux8dURX7lHqKxTt6phgoMmMpvO3lFYVGos7F13OR9NKElzjiefAQbweAt\nt8R+6A1rlIwnfywxET9ZXglfOFK6Q0nqCJhcEcKzT/Xfkd+h9XPACjOObJh3a2+o\nwg9GBFECgYEA2a6JYuFanKzvajFPbSeN1csfI9jPpK2+tB5+BB72dE74B4rjygiG\n0Rb26UjovkYfJJqKuKr4zDL5ziSlJk199Ae2f6T7t7zmyhMlWQtVT12iTQvBINTz\nNerKi5HNVBsCSGj0snbwo8u4QRgTjaIoVqTlOlUQuGqYuZ75l8g35IkCgYEAyWOM\nKagzpGmHWq/0ThN4kkwWOdujxuqrPf4un2WXsir+L90UV7X9wY4mO19pe5Ga2Upu\nXFDsxAZsanf8SbzkTGHvzUobFL7eqsiwaUSGB/cGEtkIyVlAdyW9DhiZFt3i9mEF\nbBsHnEDHPHL4tu+BB8G3WahHjWOnbWZ3NTtP94MCgYEAi3XRmSLtjYER5cPvsevs\nZ7M5oRqvdT7G9divPW6k0MEjEJn/9BjgXqbKy4ylZ/m+zBGinEsVGKXz+wjpMY/m\nCOjEGCUYC5AfgAkiHVkwb6d6asgEFEe6BaoF18MyfBbNsJxlYMzowNeslS+an1vr\nYg9EuMl06+GHNSzPlVl1zZkCgYEAxXx8N2F9eu4NUK4ZafMIGpbIeOZdHbSERp+b\nAq5yasJkT3WB/F04QXVvImv3Gbj4W7r0rEyjUbtm16Vf3sOAMTMdIHhaRCbEXj+9\nVw1eTjM8XoE8b465e92jHk6a2WSvq6IK2i9LcDvJ5QptwZ7uLjgV37L4r7sYtVx0\n69uFGJcCgYEAot7im+Yi7nNsh1dJsDI48liKDjC6rbZoCeF7Tslp8Lt+4J9CA9Ux\nSHyKjg9ujbjkzCWrPU9hkugOidDOmu7tJAxB5cS00qJLRqB5lcPxjOWcBXCdpBkO\n0tdT/xRY/MYLf3wbT95enaPlhfeqBBXKNQDya6nISbfwbMLfNxdZPJ8=\n-----END RSA PRIVATE KEY-----\n",
               "verify_client": true,
               "cipher_suites": "ECDHE-RSA-AES256-GCM-SHA384",
               "ecdh_curves": "P256"
               }`

	json.Unmarshal([]byte(test), &tlscon)
	if tlscon.ServerName != "hello.com" {
		t.Errorf("TestTlsParse failure, want hello.com but got %s", tlscon.ServerName)
	}
}

func Test_parseRouters(t *testing.T) {

	Router := []Router{
		{
			Route: RouteAction{
				TotalClusterWeight: 100,
				WeightedClusters: []WeightedCluster{
					{
						Cluster: ClusterWeight{
							Name:   "c1",
							Weight: 90,
						},
					},
					{
						Cluster: ClusterWeight{
							Name:   "c2",
							Weight: 10,
						},
					},
				},
			},
		},
	}

	if got := parseRouters(Router); len(got) != 1 || len(got[0].Route.WeightedClusters) != 2 {
		t.Errorf("parseRouters() = %v error", got)
	}
}

func Test_parseWeightClusters(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []v2.WeightedCluster
	}{
		{
			name: "validCluster",
			args: []string{
				`{ "name":"c1",	"weight":90,"metadata_match":{"filter_metadata": {"mosn.lb": {"version": "v1"}}}}`,
				`{ "name":"c2",	"weight":10,"metadata_match":{"filter_metadata": {"mosn.lb": {"version": "v2"}}}}`,
			},
			want: []v2.WeightedCluster{
				{
					Cluster: v2.ClusterWeight{
						Name:   "c1",
						Weight: 90,
						MetadataMatch: v2.Metadata{
							"version": "v1",
						},
					},
				},
				{
					Cluster: v2.ClusterWeight{
						Name:   "c2",
						Weight: 10,
						MetadataMatch: v2.Metadata{
							"version": "v2",
						},
					},
				},
			},
		},
		{
			name: "emptyCluster",
			args: []string{
				`{ "name":"c1",	"weight":90,"metadata_match":{"filter_metadata": {"mosn.lb":{} }}}`,
				`{ "name":"c2",	"weight":10,"metadata_match":{"filter_metadata": {"mosn.lb": {}}}}`,
			},
			want: []v2.WeightedCluster{
				{
					Cluster: v2.ClusterWeight{
						Name:          "c1",
						Weight:        90,
						MetadataMatch: v2.Metadata{},
					},
				},
				{
					Cluster: v2.ClusterWeight{
						Name:          "c2",
						Weight:        10,
						MetadataMatch: v2.Metadata{},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			var weightClusters []WeightedCluster
			for _, clusterString := range tt.args {
				var cluster ClusterWeight
				json.Unmarshal([]byte(clusterString), &cluster)
				weightClusters = append(weightClusters, WeightedCluster{Cluster: cluster})
			}

			if got := parseWeightClusters(weightClusters); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseWeightClusters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRouterMetadata(t *testing.T) {
	var envoyvalue = map[string]interface{}{"label": "gray", "stage": "pre-release"}
	var lbvalue = map[string]interface{}{"mosn.lb": envoyvalue}

	var envoyvalue2 = map[string]interface{}{}
	var lbvalue2 = map[string]interface{}{"mosn.lb": envoyvalue2}

	testCases := []struct {
		name string
		args map[string]interface{}
		want v2.Metadata
	}{
		{
			name: "validCase",
			args: map[string]interface{}{"filter_metadata": lbvalue},
			want: v2.Metadata{"label": "gray", "stage": "pre-release"},
		},

		{
			name: "emptyCase",
			args: map[string]interface{}{"filter_metadata": lbvalue2},
			want: v2.Metadata{},
		},
	}

	for _, tt := range testCases {
		got := parseRouterMetadata(tt.args)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parse route medata  error,want = %+v, but got = %+v, case = %s", tt.want, got, tt.name)
		}
	}
}

func TestParseTCPProxy(t *testing.T) {
	cfgStr := `{
		"routes": [
			{
				"cluster": "www",
				"source_addrs":[
					"127.0.0.1:80",
					"192.168.1.1:80"
				],
				"destination_addrs":[
					"127.0.0.1:80",
					"192.168.1.1:80"
				]
			},
			{
				"cluster": "www2",
				"source_addrs":[
					"127.0.0.1:80",
					"192.168.1.1:80"
				],
				"destination_addrs":[
					"127.0.0.1:80",
					"192.168.1.1:80"
				]
			}
		]
	}`
	cfgMap := make(map[string]interface{})
	if err := json.Unmarshal([]byte(cfgStr), &cfgMap); err != nil {
		t.Error(err)
		return
	}
	proxy, err := ParseTCPProxy(cfgMap)
	if err != nil {
		t.Error(err)
		return
	}
	addr1 := &net.TCPAddr{
		IP:   net.IPv4(127, 0, 0, 1),
		Port: 80,
	}
	addr2 := &net.TCPAddr{
		IP:   net.IPv4(192, 168, 1, 1),
		Port: 80,
	}
	route1 := &v2.TCPRoute{
		Cluster:          "www",
		SourceAddrs:      []net.Addr{addr1, addr2},
		DestinationAddrs: []net.Addr{addr1, addr2},
	}
	route2 := &v2.TCPRoute{
		Cluster:          "www2",
		SourceAddrs:      []net.Addr{addr1, addr2},
		DestinationAddrs: []net.Addr{addr1, addr2},
	}
	expected := &v2.TCPProxy{
		Routes: []*v2.TCPRoute{route1, route2},
	}
	compare := func(p1, p2 *v2.TCPProxy) bool {
		if len(p1.Routes) != len(p2.Routes) {
			return false
		}
		for i := range p1.Routes {
			r1 := p1.Routes[i]
			r2 := p2.Routes[i]
			if r1.Cluster != r2.Cluster {
				return false
			}
			if len(r1.SourceAddrs) != len(r2.SourceAddrs) {
				return false
			}
			if len(r1.DestinationAddrs) != len(r2.DestinationAddrs) {
				return false
			}
			for j := range r1.SourceAddrs {
				s1 := r1.SourceAddrs[j]
				s2 := r2.SourceAddrs[j]
				if s1.String() != s2.String() {
					return false
				}
			}
			for j := range r1.DestinationAddrs {
				d1 := r1.DestinationAddrs[j]
				d2 := r2.DestinationAddrs[j]
				if d1.String() != d2.String() {
					return false
				}
			}
		}
		return true
	}
	if !compare(expected, proxy) {
		t.Error("generate tcp proxy unexpected")
	}
}
