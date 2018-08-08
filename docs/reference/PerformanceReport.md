# 性能报告说明
以下的的性能报告为 SOFAMosn 0.1.0 在做 Bolt 与 HTTP1.x 协议的纯 TCP 转发上与 envoy 的一些性能对比数据，主要表现
在 QPS、RTT、失败率/成功率等。   
这里需要强调的是，为了提高 SOFAMosn 的转发性能，在 0.1.0 版本中，我们做了如下的一些优化手段：

+ 在线程模型优化上，使用 worker 协程池处理 stream 事件，使用两个独立的协程分别处理读写 IO
+ 在单核转发优化上，在指定 `P=1` 的情况下，我们通过使用 CPU 绑核的形式来提高系统调用的执行效率以及 cache 的 locality affinity
+ 在内存优化上，同样是在单核绑核的情况下，我们通过使用 SLAB-style 的回收机制来提高复用，减少内存 copy
+ 在 IO 优化上，主要是通过读写 buffer 大小以及读写时机和频率等参数的控制上进行调优

以下为具体的性能测试数据
# TCP 代理性能数据
这里，针对相同的部署模式，我们分别针对上层协议为 `"Bolt(SofaRpc相关协议)"` 与 `"HTTP1.1"` 来进行对比
## 部署模式 
压测采用纯代理模式部署，client 进程通过 SOFAMosn 进程作为转发代理访问server进程。其中，client 进程，
SOFAMosn 进程，server 进程分别运行在属于不同网段的机器中。client 直连访问 server 网络延时为 2.5ms 左右

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
mesh 运行在容器中，其中 CPU 为独占的一个逻辑核，具体规格如下：

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
### 性能数据

| 指标 | SOFAMosn | Envoy
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
 | 指标 | SOFAMosn | Envoy
| -------- | -------- | -------- |
| QPS      | 5600    |5600   |
| RT(mean)       | 3.549ms     |3.545ms      |
| RT(P99)       | 4ms     |4ms      |
| RT(P98)       | 4ms     |4ms      |
| RT(P95)       | 4ms     |4ms      |
| MEM      | 24m      |23m       |
| CPU      | 40%      |20%       |
 
 ### 并发40
 | 指标 | SOFAMosn | Envoy
| -------- | -------- | -------- |
| QPS      | 11150    |11200   |
| RT(mean)       | 3.583ms     |3.565ms      |
| RT(P99)       | 4ms     |4ms      |
| RT(P98)       | 4ms     |4ms      |
| RT(P95)       | 4ms     |4ms      |
| MEM      | 34m      |24m       |
| CPU      | 70%      |40%       |
 
 ### 并发200
 | 指标 | SOFAMosn | Envoy
| -------- | -------- | -------- |
| QPS      | 29670    |38800   |
| RT(mean)       | 5.715ms     |5.068ms      |
| RT(P99)       | 16ms     |7ms      |
| RT(P98)       | 13ms     |7ms      |
| RT(P95)       | 11ms     |6ms      |
| MEM      | 96m      |24m       |
| CPU      | 100%      |95%       |

 ### 并发220

| 指标 | SOFAMosn | Envoy
| -------- | -------- | -------- |
| QPS      | 30367    |41070   |
| RT(mean)       | 8.201ms     |5.369ms      |
| RT(P99)       | 20ms     |9ms      |
| RT(P98)       | 19ms     |8ms      |
| RT(P95)       | 16ms     |8ms      |
| MEM      | 100m      |24m       |
| CPU      | 100%      |100%       |

### 结论

可以看到，在上层协议为 HTTP/1.X 时，SOFAMosn 的性能和 Envoy 的性能
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