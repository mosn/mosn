# MOSN RDS 开发文档

## 概念：
MOSN 中的路由表由许多 virtual host 组成，每个 virtual host 又由许多 router 构成，每个 router 由匹配规则（Match）和对应的后端 cluster 组成。

+ 路由表数据结构：

  在具体的实现中，路由表的数据结构为：
```go
// A router wrapper used to matches an incoming request headers to a backend cluster
type RouteMatcher struct {
   virtualHosts                map[string]types.VirtualHost // key是 host(domain)
   defaultVirtualHost          types.VirtualHost  // domain = *
   wildcardVirtualHostSuffixes map[int]map[string]types.VirtualHost // *test
}
```

+ `defaultVirtualHost` 为默认使用的路由，在 `virtualHosts` 和 `wildcardVirtualHostSuffixes` 匹配失效的时候使用；
+ `virtualHosts` 为 map 结构，key 为 host 值（address 或者域名）
+ `wildcardVirtualHostSuffixes` 使用最长匹配策略，为二级索引结构，第一级索引为通配字符串长度，第二级索引为 domain 的；

## 路由表的创建：
+ 路由表在 mosn accept 连接，新建 proxy 的时候创建，实现函数：`NewRouteMatcher`

  其中：先初始化 `virtualhost`，实现函数：`NewVirtualHostImpl`
+ 在 `NewVirtualHostImpl` 中，会根据 `RouterMatch` 中的配置，选择针对 uri 中的 path，使用什么匹配策略，进而初始化具体的 router entry，新建顺序：
```go
if route.Match.Prefix != "" {
      use PrefixRouteEntryImpl
} else if route.Match.path != " {
     use PathRouteRuleImpl
}  else  if route.Match.Regex != " {
    use RegexRouteEntryImpl
} else {
    use sofa
}

```
即 path prefix 匹配 -> path 完全匹配 -> path regex 匹配 
+ 对于 http 协议族，以上三者任意可满足创建条件，但是对于 SOFA，当前的处理策略是，在上述三者均不满足的情况下，会创建 。之后，根据 `virtualHosts` 中的 `domain` 的值，初始化 `RouteMatcher`，优先级顺序以及策略如下：
```go
switch domain :
  case "*": defaultVirtualHost = virtualhost
  case "*xxx"：wildcardVirtualHostSuffixes = virtualhost
  other: virtualHosts[domain] = virtualhost

```

## 整体的匹配策略：
路由匹配的入口：
```go
func (rm *RouteMatcher) Route(headers map[string]string, randomValue uint64) types.Route
```
+ 先匹配 host（domain），获取对应的 virtual host，实现函数：`findVirtualHost`
+ 之后匹配header，支持 regex 匹配，实现函数：`ConfigUtility:: MatchHeaders`
+ 之后匹配 query parameters，支持regex，实现函数：`ConfigUtility::MatchQueryParams`
+ 之后匹配  path ，匹配优先级如下;
  + 前缀匹配，见：`PrefixRouteEntryImpl`
  + 完全匹配 ，见：`PathRouteRuleImpl`
  + 正则匹配，见：`RegexRouteEntryImpl`

## 附件
+ 当前 virtual host 的配置
```json

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
              "Value": "com.alipay.rpc.common.service.facade.pb.SampleServicePb:1.0",
              "Regex":false
            }
          ]
        },
        "Route": {
          "ClusterName": "test_cpp",
          "MetadataMatch": {
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
```