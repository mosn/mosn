# 0.2.0 架构扩展设计

## Network Filter 扩展

### 当前能力
todo
### 背景
+ 目前再 MOSN 中有三种类型的Filter： stream filter、network filter 和 listener filter
+ network filter设计上应该支持扩展，并且也有几个内置的扩展实现（proxy、tcpproxy、 fault injection）
,但是没有对应的扩展机制来保证，本次改造就是要增加network filter的扩展机制，其中stream filter的扩展机制已经存在，
network filter以此为参考进行改造

### 具体改造
+ NetworkFilterChainFactory的方法从CreateFilterFactory修改为CreateFilterChain
+ 原来调用CreateFilterFactory的地方(OnNewConnection) 修改为调用CreateFilterChain，删除buildFilterChain
+ 新增NetWorkFilterChainFactoryCallbacks 用做FilterManager的封装，明确CreateFilterChain可以调用的方法范围
+ 新增NetworkFilterFactoryCreator和filter下的Register
+ activeListener从单个NetworkFilter改为多个NetworkFilter，并且修改对应的生成函数
+ 配置解析支持多个NetworkFilter，并且将starter的逻辑移动到pkg下
