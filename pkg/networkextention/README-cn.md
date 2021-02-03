# 介绍

MOSN 在 Service Mesh 领域作为东西向 RPC 服务治理网络转发组件已经在社区很多公司都得到了一定实践，为了进一步节省研发运维等相关成本，目前业界也有很多公司在考虑统一东西、南北向数据转发面为一个。MOSN 在作为东西 Sidecar 场景当前性能足以，但在作为南北向网关会有一定性能上限（比如单机百万级长连接场景、高并发等）。

为了解决上述问题，我们在 MOSN 的 Network 层做一个高性能的网络扩展层，使其可以和高性能网络库结合使用（如Nginx、Envoy、OpenResty 等），使其底层网络具备高性能处理的同时，又能复用 MOSN Stream Filter 等能力。MOSN 通过扩展一个标准的 GOHPNF(Golang On High Performance Network Framework) 适配层和底层的高性能网络组件（如 Nginx、Envoy、Openresty 等）生态打通。使得其底层网络具备 C 同等的处理能力的同时，又能使上层业务可高效复用 MOSN 的处理能力及 GoLang 的高开发效率。


# 编译

* 编译 MOSN 以及 L7 网络扩展

执行如下命令后，会在当前路径生成 `output` 目录，该目录里面会产生两个文件 `golang_extention.so` 和 `golang_extention.h`。其中  `golang_extention.so` 是用于 MOSN 和 L7 高性能网络扩展的通信层, `golang_extention.h` 文件是提供了和 MOSN L7 场景下通信的函数声明。 

```
make build-l7
```

* 使用 Envoy 作为 MOSN 的 L7 网络扩展，并构建其 image

执行如下命令后，将会使用 `build/image/envoy` 来集成 MOSN L7 网络扩展 `golang_extention` ,编译完后将会在本地生成一个名称为 `mosn` 的 image，之后就可以启动该 image 。

```
make build-moe-image
```


# 使用事例

根据用户请求 header 的 uid 字段做路由，比如将 uid 范围在 [1,100] 的请求路由到应用的 s1 站点，将 uid 范围在 [101, 200] 的请求路由到 s2 站点，将其它范围 uid 请求路由到 s3 站点, 如果没有 uid 则将轮训转发到 s1、s2、s3。

* 步骤 1 (mosn filter 开发)

关于动态路由，已经在 MOSN 里开发了 `metadata` 模块，所以只需要在其 `metadata` 模块下扩展自己的路由实现即可。本事例我们扩展只需在该模块下扩展一个 `unit` 模块即可： 

进入 mosn/pkg/networkextention/l7/stream/filter/metadata/ ，新建 `unit/unit.go` 增加如下内容即可（只需要 70 行 Golang 就可以实现上述功能）：

```
$cat  mosn/pkg/networkextention/l7/stream/filter/metadata/unit/unit.go

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

package unit

import (
        "context"
        "encoding/json"
        "errors"
        "strconv"

        "mosn.io/api"
        "mosn.io/mosn/pkg/networkextention/l7/stream/filter/metadata"
        "mosn.io/pkg/buffer"
)

var errNotFoundUnitKey = errors.New("not found unit key")

const defaultUnitKey = "userid"

func init() {
        metadata.RegisterDriver("UNIT", &UnitDriver{})
}

type UnitConfig struct {
        UnitKey string `json:"unit_key,omitempty"`
}

type UnitDriver struct {
        unitKey string
}

func (d *UnitDriver) Init(cfg map[string]interface{}) error {
        uc := &UnitConfig{}
        data, err := json.Marshal(cfg)
        if err != nil {
                return err
        }

        if err := json.Unmarshal(data, uc); err != nil {
                return err
        }

        d.unitKey = uc.UnitKey
        // default use defaultUnitKey
        if len(d.unitKey) == 0 {
                d.unitKey = defaultUnitKey
        }

        return nil
}

func (d *UnitDriver) BuildMetadata(ctx context.Context, headers api.HeaderMap, buf buffer.IoBuffer, trailers api.HeaderMap) (string, error) {
        uid, ok := headers.Get(d.unitKey)
        if !ok || len(uid) == 0 {
                return "", errNotFoundUnitKey
        }

        if nuid, err := strconv.Atoi(uid); err == nil {
                return d.calculateMockRouter(nuid), nil
        } else {
                return "", err
        }
}

func (d *UnitDriver) calculateMockRouter(uid int) string {
        if uid >= 1 && uid <= 100 {
                return "s1"
        } else if uid >= 101 && uid <= 200 {
                return "s2"
        } else {
                return "s3"
        }
}

```

之后在 `mosn/pkg/networkextention/mosn.go` 文件末尾增加 `import  _ "mosn.io/mosn/pkg/networkextention/l7/stream/filter/metadata/unit"` 后，就可以使用 `make build-l7` or `make build-moe-image` 编译相关扩展组件了。


* 步骤 2 (修改 MOSN 及 Envoy 配置，并构建 image 其组件导入 Envoy 中 )

修改 `mosn/pkg/networkextention/build/image/envoy/conf/mosn.json` 文件，引入 UNIT 路由模块（如下 stream_filters 配置相关）：

```
{
  "servers": [
    {
      "default_log_path": "/home/admin/mosn/logs/mosn-err.log",
      "default_log_level": "WARN",
      "listeners": [
        {
          "name": "serverListener",
          "address": "0.0.0.0:2990",
          "bind_port": false,
          "log_path": "/home/admin/mosn/logs/mosn-err.log",
          "listener_filters": [],
          "stream_filters": [
            {
              "type": "metadata",
              "config": {
                "metadataers": [
                  {
                    "meta_data_key": "UNIT",
                    "config":{
                        "unit_key":"uid"
                    }
                  }
                ]
              }
            }
          ]
        }
      ]
    }
  ],
  "admin": {
    "address": {
      "socket_address": {
        "address": "0.0.0.0",
        "port_value": 34901
      }
    }
  },
  "pprof": {
    "debug": true,
    "port_value": 34902
  },
  "metrics": {
    "shm_zone": "",
    "shm_size": "0",
    "stats_matcher": {
      "exclusion_labels": [
        "host"
      ],
      "exclusion_keys": [
        "request_duration_time",
        "request_time",
        "process_time"
      ]
    },
    "sinks": [
      {
        "type": "prometheus",
        "config": {
          "port": 34903
        }
      }
    ]
  }
}

```

构建将 Envoy 作为 MOSN 的 L7 网络扩展镜像：

``
make build-moe-image
``

* 步骤 3 (运行步骤 2 构建的 image)

docker run --net=host   mosn:${MOSN-VERSION}


```
$curl localhost:2990 -H "uid:2"
site s1 from 3450


$curl localhost:2990 -H "uid:2"
site s1 from 3451


$curl localhost:2990 -H "uid:200"
site s2 from 3452


$curl localhost:2990 -H "uid:201"
site s3 from 3453

```

