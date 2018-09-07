# 0.2.0 核心能力补充
## 路由能力补充
### 支持路由时 cluster 带权重
+ [issue](https://github.com/alipay/sofa-mosn/issues/92)
+ 功能描述
  * 在当前的路由匹配逻辑中，router match 成功，会选中 router 中对应的 cluster，在这个功能中，一个
  router 允许配置多个带有权重的 cluster，在 router 匹配成功的时候，会根据 cluster 的 weight 随机返回某一个cluster
+ 配置示例
```json

                              {
                              "weighted_clusters":[
                                {
                                  "cluster":{
                                    "name":"serverCluster1",
                                    "weight":90,
                                    "metadata_match":{
                                      "filter_metadata": {
                                        "mosn.lb": {
                                          "version": "v1"
                                        }
                                      }
                                    }
                                  }
                                },
                                {
                                  "cluster":{
                                    "name":"serverCluster2",
                                    "weight":10,
                                    "metadata_match":{
                                      "filter_metadata": {
                                        "mosn.lb": {
                                          "version": "v2"
                                        }
                                      }
                                    }
                                  }
                                }
                              ]
                              }

```
## LB 能力补充
+ [issue](https://github.com/alipay/sofa-mosn/issues/91)
### 支持 host 带权重
+ 支持 host 上配置权重，用来做基于 weight 的 LB 算法
+ 配置示例
```json
        {
        "hosts": [
          {
            "address": "11.166.22.163:12200",
            "hostname": "downstream_machine1",
            "weight": 1,
            "metadata": {
              "filter_metadata": {
                "mosn.lb": {
              "stage": "pre-release",
              "version": "1.1",
              "label": "gray"
            }
              }
            }
          }
        ]
        }

```
### 支持 smooth wrr loadbalancer
+ 算法实现: `smoothWeightedRRLoadBalancer`
+ 算法细节
```cgo
/*
SW (smoothWeightedRRLoadBalancer) is a struct that contains weighted items and provides methods to select a weighted item.
It is used for the smooth weighted round-robin balancing algorithm. This algorithm is implemented in Nginx:
https://github.com/phusion/nginx/commit/27e94984486058d73157038f7950a0a36ecc6e35.
Algorithm is as follows: on each peer selection we increase current_weight
of each eligible peer by its weight, select peer with greatest current_weight
and reduce its current_weight by total number of weight points distributed
among peers.
In case of { 5, 1, 1 } weights this gives the following sequence of
current_weight's:
     a  b  c
     0  0  0  (initial state)

     5  1  1  (a selected)
    -2  1  1

     3  2  2  (a selected)
    -4  2  2

     1  3  3  (b selected)
     1 -4  3

     6 -3  4  (a selected)
    -1 -3  4

     4 -2  5  (c selected)
     4 -2 -2

     9 -1 -1  (a selected)
     2 -1 -1

     7  0  0  (a selected)
     0  0  0
*/
```

## XDS 能力补充
### CDS 相关
+ [issue](https://github.com/alipay/sofa-mosn/issues/116)
+ 提供 cluster 的添加和更新能力
   + 外部接口：`TriggerClusterAddOrUpdate`
   + 内部接口: `AddOrUpdatePrimaryCluster`
+ 提供 cluster 删除能力
   + 外部接口: `TriggerClusterDel`
   + 内部接口: `RemovePrimaryCluster`
### LDS 相关
+ [issue](https://github.com/alipay/sofa-mosn/issues/117)
+ 提供 Listener 的添加和更新能力
  + 外部接口: `AddOrUpdateListener`
  + 内部接口: `AddOrUpdateListener`
+ 提供 Listener 的删除能力
  + 外部接口: `DeleteListener`
  + 内部接口: `StopListener` 与 `RemoveListeners`