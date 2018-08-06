# 性能报告说明
以下的的性能报告为 MOSN 的 0.1.0 版本在做 Bolt 协议与 HTTP1.x 协议的纯 TCP 转发上与 envoy 的一些性能对比数据，主要表现
在 QPS、RTT、失败率/成功率等。   
这里需要强调的是，为了提高 MOSN 的转发性能，在 0.1.0 版本中，我们做了如下的一些优化手段：

+ 在线程模型优化上，使用 worker 协程池处理 stream 事件，使用两个独立的协程分别处理读写 IO
+ 在单核转发优化上，在指定 `P=1` 的情况下，我们通过使用 CPU 绑核的形式来提高系统调用的执行效率以及 cache 的 locality affinity
+ 在内存优化上，同样是在单核绑核的情况下，我们通过使用 SLAB-style 的回收机制来提高复用，减少内存 copy
+ 在 IO 优化上，主要是通过读写 buffer 大小以及读写时机和频率等参数的控制上进行调优

以下为具体的性能测试数据
# TCP 代理性能数据
这里，针对相同的部署模式，我们分别针对上层协议为 `"Bolt(SofaRpc相关协议)"` 与 `"HTTP1.1"` 来进行对比
## 部署模式
client -> mesh(11.166.161.136) -> upstream(10.210.168.5)  
其中，mesh 到 upstream 的网络延时为：1.6ms

## 客户端
### bolt 协议(发送1K字符串)
发送 Bolt 协议数据的客户端使用 "蚂蚁金服"内部开发的线上压力机，并部署 sofa rpc client。  
通过压力机的性能页面，可反映压测过程中的QPS、成功/失败次数，以及RT等参数。

### HTTP1.1 协议(发送1K字符串)
使用 ApacheBench/2.3, 测试指令:
```bash
ab -n $RPC -c $CPC -p 1k.txt -T "text/plain" -k http://11.166.161.136:12200/tcp_bench > ab.log.$CPU_IDX &
```

## mesh 运行机器规格

| 类别 | 信息 | 
| -------- | -------- | 
| OS    | 3.10.0-327.ali2008.alios7.x86_64    |
| CPU   | Intel(R) Xeon(R) CPU E5-2650 v2 @ 2.60GHz X 1 |

## upstream 运行机器规格

| 类别 | 信息 | 
| -------- | -------- | 
| OS    | 2.6.32-431.17.1.el6.FASTSOCKET    |
| CPU   | Intel(R) Xeon(R) CPU E5620  @ 2.40GHz X 16  |

## Bolt协议 测试结果
### 单核性能数据

| 指标 | MOSN | Envoy
| -------- | -------- | -------- |
| QPS      | 103500    |104000   |
| RT       | 16.23ms     |15.88ms      |
| MEM      | 31m      |18m       |
| CPU      | 100%      |100%       |

### 结论
可以看到，在单核 TCP 转发场景下，SOFAMosn 0.1.0 版本和 Envoy 1.7版本，
在满负载情况下的 QPS、RTT、成功数/失败数等性能数据上相差不大，后续版本我们会继续优化。

## HTTP/1.1 测试结果
由于 HTTP/1.1 的请求响应模型为 PING-PONG，因此 QPS 与并发数会呈现正相关。下面分别进行不同并发数的测试。
 
 ### 并发20
 | 指标 | MOSN | Envoy
| -------- | -------- | -------- |
| QPS      | 5600    |5600   |
| RT(mean)       | 3.549ms     |3.545ms      |
| RT(P99)       | 4ms     |4ms      |
| RT(P98)       | 4ms     |4ms      |
| RT(P95)       | 4ms     |4ms      |
| MEM      | 24m      |23m       |
| CPU      | 40%      |20%       |
 
 ### 并发40
 | 指标 | MOSN | Envoy
| -------- | -------- | -------- |
| QPS      | 11150    |11200   |
| RT(mean)       | 3.583ms     |3.565ms      |
| RT(P99)       | 4ms     |4ms      |
| RT(P98)       | 4ms     |4ms      |
| RT(P95)       | 4ms     |4ms      |
| MEM      | 34m      |24m       |
| CPU      | 70%      |40%       |
 
 ### 并发200
 | 指标 | MOSN | Envoy
| -------- | -------- | -------- |
| QPS      | 29670    |38800   |
| RT(mean)       | 5.715ms     |5.068ms      |
| RT(P99)       | 16ms     |7ms      |
| RT(P98)       | 13ms     |7ms      |
| RT(P95)       | 11ms     |6ms      |
| MEM      | 96m      |24m       |
| CPU      | 100%      |95%       |

 ### 并发220

| 指标 | MOSN | Envoy
| -------- | -------- | -------- |
| QPS      | 30367    |41070   |
| RT(mean)       | 8.201ms     |5.369ms      |
| RT(P99)       | 20ms     |9ms      |
| RT(P98)       | 19ms     |8ms      |
| RT(P95)       | 16ms     |8ms      |
| MEM      | 100m      |24m       |
| CPU      | 100%      |100%       |

### 结论

可以看到，在上层协议为 HTTP/1.X 时，MOSN 的性能和 Envoy 的性能
存在一定差距，对于这种现象我们的初步结论为：在 PING-PONG 的发包模型下，
MOSN无法进行 read/write 系统调用合并，相比sofarpc可以合并的场景，
syscall数量大幅上升，因此导致相比sofarpc的场景，http性能上相比envoy会存在差距。
针对这个问题，在 0.2.0 版本中，我们会进行相应的优化。

# 附录

## envoy版本信息
version:1.7
tag:1ef23d481a4701ad4a414d1ef98036bd2ed322e7


## envoy tcp测试配置

``` yaml
static_resources:
  listeners:
  - address:
      socket_address:
        address: 0.0.0.0
        port_value: 12200
    filter_chains:
    - filters:
      - name: envoy.tcp_proxy
        config:
          stat_prefix: ingress_tcp
          cluster: sofa_server
  clusters:
  - name: sofa_server
    connect_timeout: 0.25s
    type: static
    lb_policy: round_robin
    hosts:
    - socket_address:
        address: 10.210.168.5
        port_value: 12222
    - socket_address:
        address: 10.210.168.5
        port_value: 12223
    - socket_address:
        address: 10.210.168.5
        port_value: 12224
    - socket_address:
        address: 10.210.168.5
        port_value: 12225
admin:
  access_log_path: "/dev/null"
  address:
    socket_address:
      address: 0.0.0.0
      port_value: 8001
```

## MOSN sofarpc测试配置
```json
{
  "servers": [
    {
      "default_log_path": "/home/boqin/mosn/logs/default.log",
      "default_log_level": "ERROR",
      "graceful_timeout": "10s",
      "Processor": 1,
      "listeners": [
        {
          "name": "egress_http",
          "address": "0.0.0.0:12221",
          "bind_port": true,
          "filter_chains": [
            {
             "match":"",
             "tls_context":{},
              "filters": [
                {
                  "type": "proxy",
                  "config": {
                    "DownstreamProtocol": "Http1",
                    "Name": "proxy_config",
                    "SupportDynamicRoute": true,
                    "UpstreamProtocol": "Http2",
                    "VirtualHosts": [
                      {
                        "Domains": [
                          "www.antfin.com",
                          "www.alibaba.com"
                        ],
                        "Name": "www",
                        "RequireTls": "no",
                        "Routers": [
                          {
                            "Match": {
                              "Headers": [
                                {
                                  "Name": "x-custom-version",
                                  "Value": "pre-release"
                                }
                              ],
                              "Prefix": "/"
                            },
                            "Route": {
                              "ClusterName": "c1",
                              "MetadataMatch": {
                                "filter_metadata": {
                                  "envoy.lb": {
                                    "stage": "dev",
                                    "version": "1.2-pre"
                                  }
                                }
                              }
                            }
                          },
                          {
                            "Match": {
                              "Prefix": "/test/"
                            },
                            "Route": {
                              "WeightedCluster": {
                                "Clusters": [
                                  {
                                    "MetadataMatch": {
                                      "filter_metadata": {
                                        "envoy.lb": {
                                          "version": "1.0"
                                        }
                                      }
                                    },
                                    "Name": "c1",
                                    "Weight": 90
                                  },
                                  {
                                    "MetadataMatch": {
                                      "filter_metadmata": {
                                        "envoy.lb": {
                                          "version": "1.1"
                                        }
                                      }
                                    },
                                    "Name": "c2",
                                    "Weight": 10
                                  }
                                ]
                              }
                            }
                          }
                        ]
                      }
                    ]
                  }
                }
              ]
            }
          ],
          "stream_filters": [
            {
              "type": "healthcheck",
              "config": {
                "cache_time": "360s",
                "cluster_min_healthy_percentages": {
                  "local_service": 70
                },
                "passthrough": false
              }
            }
          ],
          "log_path": "/home/boqin/mosn/logs/egress.log",
          "log_level": "ERROR",
          "access_logs": [
            {
            }
          ],
          "disable_conn_io": false
        },
        {
          "name": "egress_sofa",
          "address": "0.0.0.0:12200",
          "bind_port": true,
          "filter_chains": [
            {
             "match":"",
             "tls_context":{},
              "filters": [
                {
                  "type": "proxy",
                  "config": {
                    "DownstreamProtocol": "SofaRpc",
                    "Name": "proxy_config",
                    "SupportDynamicRoute": true,
                    "UpstreamProtocol": "SofaRpc",
                    "VirtualHosts": [
                      {
                        "Name": "sofa",
                        "RequireTls": "no",
                         "Domains":[
                          "*"
                        ],
                        "Routers": [
                          {
                            "Match": {
                              "Headers": [
                                {
                                  "Name": "service",
                                  "Value": "com.alipay.test.TestService:1.0"
                                }
                              ]
                            },
                            "Route": {
                              "ClusterName": "test_cpp1",
                              "MetadataMatch": {
                                "filter_metadata": {
                                  "envoy.lb": {
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
            }
          ],
          "stream_filters": [
            {
              "type": "healthcheck",
              "config": {
                "cache_time": "360s",
                "cluster_min_healthy_percentages": {
                  "local_service": 70
                },
                "passthrough": false
              }
            }
          ],
          "log_path": "/home/boqin/mosn/logs/egress.log",
          "log_level": "ERROR",
          "disable_conn_io": false
        },
         {
          "name": "egress_sofa1",
          "address": "0.0.0.0:12202",
          "bind_port": true,
          "filter_chains": [
            {
             "match":"",
             "tls_context":{
               "status": true,
               "inspector": true,
               "server_name": "hello.com",
               "cacert": "-----BEGIN CERTIFICATE-----\nMIIDMjCCAhoCCQDaFC8PcSS5qTANBgkqhkiG9w0BAQsFADBbMQswCQYDVQQGEwJD\nTjEKMAgGA1UECAwBYTEKMAgGA1UEBwwBYTEKMAgGA1UECgwBYTEKMAgGA1UECwwB\nYTEKMAgGA1UEAwwBYTEQMA4GCSqGSIb3DQEJARYBYTAeFw0xODA2MTQwMjQyMjVa\nFw0xOTA2MTQwMjQyMjVaMFsxCzAJBgNVBAYTAkNOMQowCAYDVQQIDAFhMQowCAYD\nVQQHDAFhMQowCAYDVQQKDAFhMQowCAYDVQQLDAFhMQowCAYDVQQDDAFhMRAwDgYJ\nKoZIhvcNAQkBFgFhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArbNc\nmOXvvOqZgJdxMIiklE8lykVvz5d7QZ0+LDVG8phshq9/woigfB1aFBAI36/S5LZQ\n5Fd0znblSa+LOY06jdHTkbIBYFlxH4tdRaD0B7DbFzR5bpzLv2Q+Zf5u5RI73Nky\nH8CjW9QJjboArHkwm0YNeENaoR/96nYillgYLnunol4h0pxY7ZC6PpaB1EBaTXcz\n0iIUX4ktUJQmYZ/DFzB0oQl9IWOj18ml2wYzu9rYsySzj7EPnDOOebsRfS5hl3fz\nHi4TC4PDh0mQwHqDQ4ncztkybuRSXFQ6RzEPdR5qtp9NN/G/TlfyB0CET3AFmGkp\nE2irGoF/JoZXEDeXmQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQApzhQLS7fAcExZ\nx1S+hcy7lLF8QcPlsiH32SnLFg5LPy4prz71mebUchmt97t4T3tSWzwXi8job7Q2\nONYc6sr1LvaFtg7qoCfz5fPP5x+kKDkEPwCDJSTVPcXP+UtA407pxX8KPRN8Roay\ne3oGcmNqVu/DkkufkIL3PBg41JEMovWtKD+PXmeBafc4vGCHSJHJBmzMe5QtwHA0\nss/A9LHPaq3aLcIyFr8x7clxc7zZVaim+lVfNV3oPBnB4gU7kLFVT0zOhkM+V1A4\nQ5GVbGAu4R7ItY8kJ2b7slre0ajPUp2FMregt4mnUM3mu1nbltVhtoknXqHHMGgN\n4Lh4JfNx\n-----END CERTIFICATE-----\n",
               "certchain": "-----BEGIN CERTIFICATE-----\nMIIDJTCCAg0CAQEwDQYJKoZIhvcNAQELBQAwWzELMAkGA1UEBhMCQ04xCjAIBgNV\nBAgMAWExCjAIBgNVBAcMAWExCjAIBgNVBAoMAWExCjAIBgNVBAsMAWExCjAIBgNV\nBAMMAWExEDAOBgkqhkiG9w0BCQEWAWEwHhcNMTgwNjE0MDMxMzQyWhcNMTkwNjE0\nMDMxMzQyWjBWMQswCQYDVQQGEwJDTjEKMAgGA1UECAwBYTEKMAgGA1UECgwBYTEK\nMAgGA1UECwwBYTERMA8GA1UEAwwIdGVzdC5jb20xEDAOBgkqhkiG9w0BCQEWAWEw\nggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCrPq+Mo0nS3dJU1qGFwlIB\ni9HqRm5RGcfps+0W5LjEhqUKxKUweRrwDaIxpiSqjKeehz9DtLUpXBD29pHuxODU\nVsMH2U1AGWn9l4jMnP6G5iTMPJ3ZTXszeqALe8lm/f807ZA0C7moc+t7/d3+b6d2\nlnwR+yWbIZJUu2qw+HrR0qPpNlBP3EMtlQBOqf4kCl6TfpqrGfc9lW0JjuE6Taq3\ngSIIgzCsoUFe30Yemho/Pp4zA9US97DZjScQLQAGiTsCRDBASxXGzODQOfZL3bCs\n2w//1lqGjmhp+tU1nR4MRN+euyNX42ioEz111iB8y0VzuTIsFBWwRTKK1SF7YSEb\nAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABnRM9JJ21ZaujOTunONyVLHtmxUmrdr\n74OJW8xlXYEMFu57Wi40+4UoeEIUXHviBnONEfcITJITYUdqve2JjQsH2Qw3iBUr\nmsFrWS25t/Krk2FS2cKg8B9azW2+p1mBNm/FneMv2DMWHReGW0cBp3YncWD7OwQL\n9NcYfXfgBgHdhykctEQ97SgLHDKUCU8cPJv14eZ+ehIPiv8cDWw0mMdMeVK9q71Y\nWn2EgP7HzVgdbj17nP9JJjNvets39gD8bU0g2Lw3wuyb/j7CHPBBzqxh+a8pihI5\n3dRRchuVeMQkMuukyR+/A8UrBMA/gCTkXIcP6jKO1SkKq5ZwlMmapPc=\n-----END CERTIFICATE-----\n",
               "privateKey": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAqz6vjKNJ0t3SVNahhcJSAYvR6kZuURnH6bPtFuS4xIalCsSl\nMHka8A2iMaYkqoynnoc/Q7S1KVwQ9vaR7sTg1FbDB9lNQBlp/ZeIzJz+huYkzDyd\n2U17M3qgC3vJZv3/NO2QNAu5qHPre/3d/m+ndpZ8EfslmyGSVLtqsPh60dKj6TZQ\nT9xDLZUATqn+JApek36aqxn3PZVtCY7hOk2qt4EiCIMwrKFBXt9GHpoaPz6eMwPV\nEvew2Y0nEC0ABok7AkQwQEsVxszg0Dn2S92wrNsP/9Zaho5oafrVNZ0eDETfnrsj\nV+NoqBM9ddYgfMtFc7kyLBQVsEUyitUhe2EhGwIDAQABAoIBAG2Bj5ca0Fmk+hzA\nh9fWdMSCWgE7es4n81wyb/nE15btF1t0dsIxn5VE0qR3P1lEyueoSz+LrpG9Syfy\nc03B3phKxzscrbbAybOeFJ/sASPYxk1IshRE5PT9hJzzUs6mvG1nQWDW4qmjP0Iy\nDKTpV6iRANQqy1iRtlay5r42l6vWwHfRfwAv4ExSS+RgkYcavqOp3e9If2JnFJuo\n7Zds2i7Ux8dURX7lHqKxTt6phgoMmMpvO3lFYVGos7F13OR9NKElzjiefAQbweAt\nt8R+6A1rlIwnfywxET9ZXglfOFK6Q0nqCJhcEcKzT/Xfkd+h9XPACjOObJh3a2+o\nwg9GBFECgYEA2a6JYuFanKzvajFPbSeN1csfI9jPpK2+tB5+BB72dE74B4rjygiG\n0Rb26UjovkYfJJqKuKr4zDL5ziSlJk199Ae2f6T7t7zmyhMlWQtVT12iTQvBINTz\nNerKi5HNVBsCSGj0snbwo8u4QRgTjaIoVqTlOlUQuGqYuZ75l8g35IkCgYEAyWOM\nKagzpGmHWq/0ThN4kkwWOdujxuqrPf4un2WXsir+L90UV7X9wY4mO19pe5Ga2Upu\nXFDsxAZsanf8SbzkTGHvzUobFL7eqsiwaUSGB/cGEtkIyVlAdyW9DhiZFt3i9mEF\nbBsHnEDHPHL4tu+BB8G3WahHjWOnbWZ3NTtP94MCgYEAi3XRmSLtjYER5cPvsevs\nZ7M5oRqvdT7G9divPW6k0MEjEJn/9BjgXqbKy4ylZ/m+zBGinEsVGKXz+wjpMY/m\nCOjEGCUYC5AfgAkiHVkwb6d6asgEFEe6BaoF18MyfBbNsJxlYMzowNeslS+an1vr\nYg9EuMl06+GHNSzPlVl1zZkCgYEAxXx8N2F9eu4NUK4ZafMIGpbIeOZdHbSERp+b\nAq5yasJkT3WB/F04QXVvImv3Gbj4W7r0rEyjUbtm16Vf3sOAMTMdIHhaRCbEXj+9\nVw1eTjM8XoE8b465e92jHk6a2WSvq6IK2i9LcDvJ5QptwZ7uLjgV37L4r7sYtVx0\n69uFGJcCgYEAot7im+Yi7nNsh1dJsDI48liKDjC6rbZoCeF7Tslp8Lt+4J9CA9Ux\nSHyKjg9ujbjkzCWrPU9hkugOidDOmu7tJAxB5cS00qJLRqB5lcPxjOWcBXCdpBkO\n0tdT/xRY/MYLf3wbT95enaPlhfeqBBXKNQDya6nISbfwbMLfNxdZPJ8=\n-----END RSA PRIVATE KEY-----\n",
               "verifyClient": true,
               "cipherSuites": "ECDHE-RSA-AES256-GCM-SHA384",
               "ecdhCurves": "P256"
               },
              "filters": [
                {
                  "type": "proxy",
                  "config": {
                    "DownstreamProtocol": "SofaRpc",
                    "Name": "proxy_config",
                    "SupportDynamicRoute": true,
                    "UpstreamProtocol": "SofaRpc",
                    "VirtualHosts": [
                      {
                        "Name": "sofa",
                        "RequireTls": "no",
                         "Domains":[
                          "*"
                        ],
                        "Routers": [
                          {
                            "Match": {
                              "Headers": [
                                {
                                  "Name": "service",
                                  "Value": "com.alipay.rpc.common.service.facade.pb.SampleServicePb:1.0"
                                }
                              ]
                            },
                            "Route": {
                              "ClusterName": "test_cpp",
                              "MetadataMatch": {
                                "filter_metadata": {
                                  "envoy.lb": {
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
            }
          ],
          "stream_filters": [
            {
              "type": "healthcheck",
              "config": {
                "cache_time": "360s",
                "cluster_min_healthy_percentages": {
                  "local_service": 70
                },
                "passthrough": false
              }
            }
          ],
          "log_path": "/home/boqin/mosn/logs/egress.log",
		  "access_logs":[
			{
			}
		],
          "disable_conn_io": false
        },
        {
          "name": "ingress_sofa",
          "address": "127.0.0.1:8889",
          "bind_port": true,
          "filter_chains": [
            {
             "match":"",
             "tls_context":{},
              "Filters": [
                {
                  "type": "proxy",
                  "config": {
                    "DownstreamProtocol": "SofaRpc",
                    "Name": "proxy_config",
                    "SupportDynamicRoute": true,
                    "UpstreamProtocol": "SofaRpc",
                    "VirtualHosts": [
                      {
                        "Name": "sofa",
                        "RequireTls": "no",
                        "Routers": [
                          {
                            "Match": {
                              "Headers": [
                                {
                                  "Name": "service",
                                  "Value": "com.alipay.rpc.common.service.facade.pb.SampleServicePb:1.0"
                                }
                              ]
                            },
                            "Route": {
                              "ClusterName": "c1",
                              "MetadataMatch": {
                                "filter_metadata": {
                                  "envoy.lb": {
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
            }
          ],
          "stream_filters": [
            {
              "type": "healthcheck",
              "config": {
                "cache_time": "360s",
                "cluster_min_healthy_percentages": {
                  "local_service": 70
                },
                "passthrough": false
              }
            }
          ],
          "log_path": "/home/boqin/mosn/logs/egress.log",
          "log_level": "ERROR",
          "access_logs": [
            {
              "log_path": "./access_egress.log",
              "log_format": "%StartTime% %RequestReceivedDuration% %ResponseReceivedDuration% %REQ.requestid% %REQ.cmdcode% %RESP.requestid% %RESP.service%"
            }
          ],
          "disable_conn_io": false
        }
      ]
    }
  ],
  "cluster_manager": {
    "auto_discovery": true,
    "clusters": [
      { "Name": "test_cpp",
        "Type": "SIMPLE",
        "sub_type": "",
        "lb_type": "LB_RANDOM",
        "MaxRequestPerConn": 0,
        "ConnBufferLimitBytes": 0,
        "circuit_breakers":[
          {
            "priority":"default",
          "max_connections": 10240,
          "max_pending_requests": 10240,
          "max_requests": 10240,
          "max_retries": 3
        }
      ],

        "health_check": {
		"protocol":"SofaRpc",
          "Timeout": "90s",
          "HealthyThreshold": 2,
          "UnhealthyThreshold": 2,
          "Interval":"5s",
          "IntervalJitter": 0,
          "CheckPath": "",
          "ServiceName": ""
        },
        "spec": {},
        "hosts": [
          {
            "Address": "11.166.22.163:12200",
            "Hostname": "downstream_machine1",
            "Weight": 1,
            "MetaData": {
              "label": "gray",
              "stage": "1"
            }
          }
        ],
        "LBSubsetConfig": {
          "FallBackPolicy": 0,
          "DefaultSubset": null,
          "SubsetSelectors": null
        }
      },
      { "Name": "test_cpp1",
        "Type": "SIMPLE",
        "sub_type": "",
        "lb_type": "LB_RANDOM",
        "health_check": {
		"protocol":"SofaRpc",
          "Timeout": "90s",
          "HealthyThreshold": 2,
          "UnhealthyThreshold": 2,
          "Interval":"5s",
          "IntervalJitter": 0,
          "CheckPath": "",
          "ServiceName": ""
        },

        "spec": {},
        "tls_context":{
          "status": false,
          "server_name": "hello.com",
          "cacert": "-----BEGIN CERTIFICATE-----\nMIIDMjCCAhoCCQDaFC8PcSS5qTANBgkqhkiG9w0BAQsFADBbMQswCQYDVQQGEwJD\nTjEKMAgGA1UECAwBYTEKMAgGA1UEBwwBYTEKMAgGA1UECgwBYTEKMAgGA1UECwwB\nYTEKMAgGA1UEAwwBYTEQMA4GCSqGSIb3DQEJARYBYTAeFw0xODA2MTQwMjQyMjVa\nFw0xOTA2MTQwMjQyMjVaMFsxCzAJBgNVBAYTAkNOMQowCAYDVQQIDAFhMQowCAYD\nVQQHDAFhMQowCAYDVQQKDAFhMQowCAYDVQQLDAFhMQowCAYDVQQDDAFhMRAwDgYJ\nKoZIhvcNAQkBFgFhMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEArbNc\nmOXvvOqZgJdxMIiklE8lykVvz5d7QZ0+LDVG8phshq9/woigfB1aFBAI36/S5LZQ\n5Fd0znblSa+LOY06jdHTkbIBYFlxH4tdRaD0B7DbFzR5bpzLv2Q+Zf5u5RI73Nky\nH8CjW9QJjboArHkwm0YNeENaoR/96nYillgYLnunol4h0pxY7ZC6PpaB1EBaTXcz\n0iIUX4ktUJQmYZ/DFzB0oQl9IWOj18ml2wYzu9rYsySzj7EPnDOOebsRfS5hl3fz\nHi4TC4PDh0mQwHqDQ4ncztkybuRSXFQ6RzEPdR5qtp9NN/G/TlfyB0CET3AFmGkp\nE2irGoF/JoZXEDeXmQIDAQABMA0GCSqGSIb3DQEBCwUAA4IBAQApzhQLS7fAcExZ\nx1S+hcy7lLF8QcPlsiH32SnLFg5LPy4prz71mebUchmt97t4T3tSWzwXi8job7Q2\nONYc6sr1LvaFtg7qoCfz5fPP5x+kKDkEPwCDJSTVPcXP+UtA407pxX8KPRN8Roay\ne3oGcmNqVu/DkkufkIL3PBg41JEMovWtKD+PXmeBafc4vGCHSJHJBmzMe5QtwHA0\nss/A9LHPaq3aLcIyFr8x7clxc7zZVaim+lVfNV3oPBnB4gU7kLFVT0zOhkM+V1A4\nQ5GVbGAu4R7ItY8kJ2b7slre0ajPUp2FMregt4mnUM3mu1nbltVhtoknXqHHMGgN\n4Lh4JfNx\n-----END CERTIFICATE-----\n",
          "certchain": "-----BEGIN CERTIFICATE-----\nMIIDJTCCAg0CAQEwDQYJKoZIhvcNAQELBQAwWzELMAkGA1UEBhMCQ04xCjAIBgNV\nBAgMAWExCjAIBgNVBAcMAWExCjAIBgNVBAoMAWExCjAIBgNVBAsMAWExCjAIBgNV\nBAMMAWExEDAOBgkqhkiG9w0BCQEWAWEwHhcNMTgwNjE0MDMxMzQyWhcNMTkwNjE0\nMDMxMzQyWjBWMQswCQYDVQQGEwJDTjEKMAgGA1UECAwBYTEKMAgGA1UECgwBYTEK\nMAgGA1UECwwBYTERMA8GA1UEAwwIdGVzdC5jb20xEDAOBgkqhkiG9w0BCQEWAWEw\nggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAwggEKAoIBAQCrPq+Mo0nS3dJU1qGFwlIB\ni9HqRm5RGcfps+0W5LjEhqUKxKUweRrwDaIxpiSqjKeehz9DtLUpXBD29pHuxODU\nVsMH2U1AGWn9l4jMnP6G5iTMPJ3ZTXszeqALe8lm/f807ZA0C7moc+t7/d3+b6d2\nlnwR+yWbIZJUu2qw+HrR0qPpNlBP3EMtlQBOqf4kCl6TfpqrGfc9lW0JjuE6Taq3\ngSIIgzCsoUFe30Yemho/Pp4zA9US97DZjScQLQAGiTsCRDBASxXGzODQOfZL3bCs\n2w//1lqGjmhp+tU1nR4MRN+euyNX42ioEz111iB8y0VzuTIsFBWwRTKK1SF7YSEb\nAgMBAAEwDQYJKoZIhvcNAQELBQADggEBABnRM9JJ21ZaujOTunONyVLHtmxUmrdr\n74OJW8xlXYEMFu57Wi40+4UoeEIUXHviBnONEfcITJITYUdqve2JjQsH2Qw3iBUr\nmsFrWS25t/Krk2FS2cKg8B9azW2+p1mBNm/FneMv2DMWHReGW0cBp3YncWD7OwQL\n9NcYfXfgBgHdhykctEQ97SgLHDKUCU8cPJv14eZ+ehIPiv8cDWw0mMdMeVK9q71Y\nWn2EgP7HzVgdbj17nP9JJjNvets39gD8bU0g2Lw3wuyb/j7CHPBBzqxh+a8pihI5\n3dRRchuVeMQkMuukyR+/A8UrBMA/gCTkXIcP6jKO1SkKq5ZwlMmapPc=\n-----END CERTIFICATE-----\n",
          "privateKey": "-----BEGIN RSA PRIVATE KEY-----\nMIIEpQIBAAKCAQEAqz6vjKNJ0t3SVNahhcJSAYvR6kZuURnH6bPtFuS4xIalCsSl\nMHka8A2iMaYkqoynnoc/Q7S1KVwQ9vaR7sTg1FbDB9lNQBlp/ZeIzJz+huYkzDyd\n2U17M3qgC3vJZv3/NO2QNAu5qHPre/3d/m+ndpZ8EfslmyGSVLtqsPh60dKj6TZQ\nT9xDLZUATqn+JApek36aqxn3PZVtCY7hOk2qt4EiCIMwrKFBXt9GHpoaPz6eMwPV\nEvew2Y0nEC0ABok7AkQwQEsVxszg0Dn2S92wrNsP/9Zaho5oafrVNZ0eDETfnrsj\nV+NoqBM9ddYgfMtFc7kyLBQVsEUyitUhe2EhGwIDAQABAoIBAG2Bj5ca0Fmk+hzA\nh9fWdMSCWgE7es4n81wyb/nE15btF1t0dsIxn5VE0qR3P1lEyueoSz+LrpG9Syfy\nc03B3phKxzscrbbAybOeFJ/sASPYxk1IshRE5PT9hJzzUs6mvG1nQWDW4qmjP0Iy\nDKTpV6iRANQqy1iRtlay5r42l6vWwHfRfwAv4ExSS+RgkYcavqOp3e9If2JnFJuo\n7Zds2i7Ux8dURX7lHqKxTt6phgoMmMpvO3lFYVGos7F13OR9NKElzjiefAQbweAt\nt8R+6A1rlIwnfywxET9ZXglfOFK6Q0nqCJhcEcKzT/Xfkd+h9XPACjOObJh3a2+o\nwg9GBFECgYEA2a6JYuFanKzvajFPbSeN1csfI9jPpK2+tB5+BB72dE74B4rjygiG\n0Rb26UjovkYfJJqKuKr4zDL5ziSlJk199Ae2f6T7t7zmyhMlWQtVT12iTQvBINTz\nNerKi5HNVBsCSGj0snbwo8u4QRgTjaIoVqTlOlUQuGqYuZ75l8g35IkCgYEAyWOM\nKagzpGmHWq/0ThN4kkwWOdujxuqrPf4un2WXsir+L90UV7X9wY4mO19pe5Ga2Upu\nXFDsxAZsanf8SbzkTGHvzUobFL7eqsiwaUSGB/cGEtkIyVlAdyW9DhiZFt3i9mEF\nbBsHnEDHPHL4tu+BB8G3WahHjWOnbWZ3NTtP94MCgYEAi3XRmSLtjYER5cPvsevs\nZ7M5oRqvdT7G9divPW6k0MEjEJn/9BjgXqbKy4ylZ/m+zBGinEsVGKXz+wjpMY/m\nCOjEGCUYC5AfgAkiHVkwb6d6asgEFEe6BaoF18MyfBbNsJxlYMzowNeslS+an1vr\nYg9EuMl06+GHNSzPlVl1zZkCgYEAxXx8N2F9eu4NUK4ZafMIGpbIeOZdHbSERp+b\nAq5yasJkT3WB/F04QXVvImv3Gbj4W7r0rEyjUbtm16Vf3sOAMTMdIHhaRCbEXj+9\nVw1eTjM8XoE8b465e92jHk6a2WSvq6IK2i9LcDvJ5QptwZ7uLjgV37L4r7sYtVx0\n69uFGJcCgYEAot7im+Yi7nNsh1dJsDI48liKDjC6rbZoCeF7Tslp8Lt+4J9CA9Ux\nSHyKjg9ujbjkzCWrPU9hkugOidDOmu7tJAxB5cS00qJLRqB5lcPxjOWcBXCdpBkO\n0tdT/xRY/MYLf3wbT95enaPlhfeqBBXKNQDya6nISbfwbMLfNxdZPJ8=\n-----END RSA PRIVATE KEY-----\n",
          "cipherSuites": "ECDHE-RSA-AES256-GCM-SHA384",
          "ecdhCurves": "P256"
          },
        "hosts": [
          {
            "Address": "127.0.0.1:12222",
            "Hostname": "downstream_machine1",
            "Weight": 1,
            "MetaData": {
              "label": "gray",
              "stage": "1"
            }
          },
		  {
            "Address": "127.0.0.1:12223",
            "Hostname": "downstream_machine2",
            "Weight": 1,
            "MetaData": {
              "label": "gray",
              "stage": "1"
            }
          },
		  {
            "Address": "127.0.0.1:12224",
            "Hostname": "downstream_machine3",
            "Weight": 1,
            "MetaData": {
              "label": "gray",
              "stage": "1"
            }
          },
		  {
            "Address": "127.0.0.1:12225",
            "Hostname": "downstream_machine4",
            "Weight": 1,
            "MetaData": {
              "label": "gray",
              "stage": "1"
            }
          },
		  {
            "Address": "127.0.0.1:12226",
            "Hostname": "downstream_machine5",
            "Weight": 1,
            "MetaData": {
              "label": "gray",
              "stage": "1"
            }
          }
        ],
        "LBSubsetConfig": {
          "FallBackPolicy": 0,
          "DefaultSubset": null,
          "SubsetSelectors": null
        }
      }
    ]
  },
  "service_registry": {
    "application": {
      "ant_share_cloud": false,
      "data_center": "dc",
      "app_name": "testapp"
    }
  }
}
```