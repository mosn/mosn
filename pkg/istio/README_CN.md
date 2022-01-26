## 背景

基于最新的MOSN架构（v0.24.0），我们将MOSN对接Istio相关的能力从MOSN核心框架能力中进行了解耦合。MOSN核心框架中只包含一些抽象的行为，不涉及具体Istio 版本实现，具体实现中主要涉及的内容就是go-control-plane相关的部分。
部分扩展功能也可以考虑做一层抽象，将istio相关部分解耦，减少不同istio版本对接中代码的重复，目前暂时忽略这部分，将这部分扩展整体作为具体实现的一部分，从MOSN核心框架中分离。

## 详细实现方案

一个Istio版本实现的部分，可以划分为三大模块：
- xDS 模块，用于对接Istio配置转换为MOSN配置。xDS模块是对接Istio版本必须实现的模块
- main 模块，用于将Istio具体实现和MOSN框架结合，生成支持对应Istio版本。main模块是对接Istio版本必须实现的模块
- 扩展模块，用于实现一些和Istio版本相关的扩展部分。扩展模块不是必须实现的模块，对接不同的Istio版本涉及的内容也不尽相同。
- 相关脚本，非必须实现，但是建议实现。

详细的方案说明可以参考 [mosn#1744](https://github.com/mosn/mosn/pull/1744)

### xDS 模块

- 需要实现`mosn.io/mosn/pkg/istio.XdsStreamConfig`和`mosn.io/mosn/pkg/istio.XdsStreamClient`，这两个实现包含了绝大部分`xds`适配相关的逻辑，使用`mosn.io/mosn/pkg/istio.RegisterParseAdsConfig`注册完成扩展。
- 需要实现`mosn.io/mosn/pkg/mtls/sds.SdsStreamClient`，包含了sds相关的逻辑，使用`mosn.io/mosn/pkg/mtls/sds.RegisterSdsStreamClientFactory`注册完成扩展。

### main 模块

- main实现，主要包含istio 版本相关的`xDS模块`和`扩展模块`的引用，其余部分存在一定的通用性，后续改造过程中计划抽象出去。

### 扩展模块

扩展模块没有必须实现的部分，一般来说是根据实际的需求进行对应的实现，通常来说，主要是MOSN核心框架中提供的一些通用的扩展能力，如
- Stream Filter扩展，如果Stream Filter的实现中包含和Istio 版本相关的代码，建议和Istio 对接版本绑定。如果存在一定的通用逻辑，可以将抽象部分写在MOSN 核心框架中，减少代码的重复性。
- `mosn.io/mosn/pkg/istio.XdsInfo`相关的赋值扩展，通常来说不同的Istio版本也会有不同的需求
- `mosn.io/mosn/pkg/cel` 的默认属性扩展。
- 其他根据实际需求进行的扩展。

### 相关脚本

- `Makefile`，便于快速调整istio版本对接情况的脚本，非必须实现，但是建议实现。通常建议将最新的Istio版本作为默认的版本。
- 集成测试相关，在`etc/script/report.sh`中追加对应的运行测试脚本，通常来说预期应该是确保多个版本兼容。
- `ISTIO_VERSION`修改为对应的版本号


## 实际案例（Istio v1.5.2）

- 在MOSN 目录下，新增了一个`istio/istio152`目录，目录中包含对接Istio v1.5.2 相关的所有代码，后续其他版本也建议使用类似的目录方便快速查找`istio/istio${version}`
- xDS模块实现：`istio/istio152/xds`和`istio/istio152/sds`
- main模块实现：`istio/istio152/main`
  - istio v1.5.2 相关的一些启动参数设置
  - 引用istio v1.5.2相关的内容，确保对应的`func init()`被正确执行
- 扩展模块实现
  - `istio/istio152/filter`、`istio/istio152/istio`、`istio/istio152/config`，包含istio 版本相关的stream filter和其他部分的实现，其中部分代码具备进一步抽象到MOSN核心框架的能力，后续考虑实现
  - `istio/istio152/cel.go` 包含MOSN中CEL模块默认属性的扩展，如果没有这个扩展，CEL模块支持的属性会少一些，可根据实际需求调整
  - `istio/istio152/info.go` 设置xdsInfo。
- 相关脚本
  - `Makefile` 中新增了 `istio-1.5.2`和`unit-test-istio-1.5.2`，其中`istio-1.5.2`会将默认的支持的istio版本调整为v1.5.2。
  - 在`etc/script/report.sh`中追加对应的运行测试命令。
- 当前默认的主干中，是执行了`make istio-1.5.2`的结果
  - `ISTIO_VERSION`修改为了1.5.2
