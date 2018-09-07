# dubbogo #
a golang micro-service framework compatible with alibaba dubbo. just using jsonrpc 2.0 protocol over http now.

## 说明 ##
---
> dubbogo 目前版本(0.1.1)支持的codec 是jsonrpc 2.0，transport protocol是http。
> 只要你的java程序支持jsonrpc 2.0 over http，那么dubbogo程序就能调用它。
> dubbogo自己的server端也已经实现，即dubbogo既能调用java service也能调用dubbogo实现的service。


## develop list ##
---

### 2018-05-17
---
- 1 把github.com/AlexStocks/gohessian最新的hessian2解析代码合并到dubbogo/codec/hessian下面；
- 2 改进tcp transport socket字节流读取机制；
- 3 registry/config.go:ServiceConfig::ServiceEqual 严格过滤条件。

### 2018-05-07
---
- 1 fmt.Errorf -> juju/errors.Errorf
- 2 delete ListService
- 3 Registry.GetService -> Registry.GetServices
- 4 use ServiceConfigIf in selector.Select()
- 5 把dubbog-examples client下面一些默认配置放到client package 下面
- 6 把hessian codec和rpc client对接起来

### 2018-04-19

------

- 1 bug fix: registry/zk/server.go:handleZkRestart() 把services从函数开头移动到重复注册的for-loop外面，以防止 zk 不断重启时，这个数组可能有重复元素的问题；

## 2017-10-05

* 1 dubbogo/codec/hessian添加 go int +struct 对应 java enum；

### 2016-10-26 ###
---
- 1 添加 github.com/AlexStocks/dubbogo-examples/calculator/java-server 作为client_test.go的mock服务端

### 2016-10-23 ###
---
- 1 把所有有关ID的field的type由uint64改为int64，因为java没有uint64类型 

###  ###
### 2016-10-22 ###
---
- 1 modify registry/zk/client.go:(consumerZookeeperRegistry)register to normalize zookeeper consumer url
- 2 modify registry/zk/server.go:(providerZookeeperRegistry)register to normalize zookeeper provider url
- 3 修改 server/server.go:(server)Handle 函数，增加了监察@Handle是否注册成功的逻辑

### 2016-10-19 ###
---
- 1 调整dubbogo/server下几个文件的名称，并且让server和rpcServer两个struct交换名称，以消除源码名称与其包含的struct名称不一致带来的歧义;

### 2016-10-17 ###
---
- 1 修改dubbogo/zk/registry.go中DubboRole的定义(suggestion from 包增辉)
    > zk/registry.go中DubboRole的定义应改成：
    >
    > DubboRole  = [...]string{"consumers", "", "", "providers"}
    >
    > 否则，dubbo java consumer来启动找provider url时，因为category不匹配，会找不到provider，导致consumer启动不了。

### 2016-08-18 ###
---
- 1 基于go1.7，修正package context的路径；

### 2016-08-12 ###
---
- 1 测试过程中发现，dubbogo/registry/zk/registry.go:(zookeeperRegistry)validateZookeeperClient()之后使用了RegistryZkClient,修改如下代码：
    * a dubbogo/registry/zk/client.go:NewConsumerZookeeperRegistry()中调用newZookeeperRegistry后添加clause “reg.client.name = ConsumerRegistryZkClient";
    * b dubbogo/registry/zk/server.go:NewProviderZookeeperRegistry()中调用newZookeeperRegistry后添加clause “reg.client.name = ProviderRegistryZkClient";
- 2 修正dubbogo/registry/zk/client.go和dubbogo/registry/zk/server.go中的同样处理zk重启事件的函数handleZkRestart中的相关逻辑
   此处须注意zk重启后会load之前的数据，导致启动之前注册的tmp节点数据又被load进来，大概10s之后才会被zk删除掉。
- 3 修正dubbogo/transport/http_transport.go中有关httpTransport成员相关的grammar error

### 2016-08-11 ###
---
- 1 测试过程中发现，如果一个zkPath下的node为空，而后添加一个node的时候，dubbo/registry/zk/watch.go:(zookeeperWatcher)watchDir()
    函数中for-select clause中time.After会导致watcher不能及时收到相关的event，如果剔除time.After又会导致疯狂for-select以致于影响程序性能，现做以下修改:

    * a dubbo/registry/zk/zk_client.go中添加zookeeperClient{entryRegistry}以及相关函数;
    * b dubbo/registry/zk/watch.go:(zookeeperWatcher)watchDir() for-select添加一个case分支，以处理zkClient发来的event;

    所以发生所有服务死掉的情况时，dubbogo的服务发现过程为:

    * a 第一个service的发现，要依靠dubbogo/registry/zk:(zookeeperClient) handleZkEvent()通知dubbogo/registry/zk:(zookeeperWatcher) watchDir()；
    * b 第二个service启动后，dubbogo/registry/zk:(zookeeperWatcher) watchDir()自身能否及时发现值。

- 2 测试过程发现dubbogo/registry/zk:zookeeperClient) handleZkEvent()已经能够应对zk启停(死掉重启)的情况，所以注释掉这两个函数:

    > a dubbogo/registry/zk:(consumerZookeeperRegistry)reconnectZkRegister()
    > b dubbogo/registry/zk:(providerZookeeperRegistry)reconnectZkRegister()

### 2016-08-10 ###
---
- 1 dubbogo/selector/cache/cache.go:(cacheSelector)watch中添加一个内部exit chan，用于watch频繁创建的时候，让这个函数的内部goroutine退出;
- 2 dubbogo/registry/zk/watch.go:zookeeperWatcher添加once成员，以在(zookeeperWatcher)Stop函数中只执行一次client.Stop
- 3 dubbogo/registry/zk/client.go:(consumerZookeeperRegistry)Watch中，把for循环中的zkWatcher.watchService另起一个goroutine进行异步化操作，以防止阻塞for-loop。
- 4 dubbogo/registry/zk/watch.go:(zookeeperWatcher)watchService中，注释掉给this.events发送error的代码

     > 不要发送不必要的error给selector，以防止selector/cache/cache.go:(cacheSelector)watch，调用(zookeeperWatcher)Next获取error后，不断退出

- 5 为了dubbogo/registry/client与registry连接断开的情况下，dubbogo/selector能继续稳定地提供服务，修改：

     * a dubbogo/registry/zk/watch.go:(zookeeperWatcher)watchDir()里面的watchServiceNode goroutine;
     * b dubbogo/registry/zk/watch.go:(zookeeperWatcher)watchService()里面的watchServiceNode goroutine;
     * c dubbogo/registry/zk/watch.go:(zookeeperWatcher)watchService()添加Valid method;
     * d dubbogo/selector/cache.go:(cacheSelector) watch()中的for循环时收到delete event时检查watcher.Valid;

### 2016-08-08 ###
---
- 1 考虑到dubbogo/errs主要由dubbogo/client/rpc_client.go使用，把dubbogo/errs/errors.go挪到common之中;
- 2 修正dubbogo/selector/mode.go:roundRobin算法，给dubbogo/selector下的Next函数添加一个入参ID uint64

     > 因为selector仅仅返回了一个serviceURL array，如果不传入一个参数，每次roundRobin都从0开始，会导致只使用第一个地址

### 2016-08-07 ###
---
- 1 经过2016-08-06#2的排查，查明了(rpcClient)call的poolConn.Close()&rpcStream.Close()两处close逻辑有冲突，做以下修改:

     a: 注释掉poolConn.Close处的代码;
     b: 给poolConn添加once成员，确保只释放一次;
     c: 注意将来添加长连接逻辑时候，此处相关逻辑可能还要修改;
     相关的error为：
     >> [08/05/16 20:02:37] [DEBG] @i{0}, call(Id{3979924209}, ctx{context.Background.WithValue("dubbogo-ctx", map[string]string{"X-Proxy-Id":"dubbogo", "X-Services":"im.youni.iccs.status.api.StatusUpdateService", "X-Method":"login0"}).WithDeadline(2016-08-05 20:02:37.478436456 +0800 CST [149.341097ms])}, address{116.211.15.189:30880}, path{/im.youni.iccs.status.api.StatusUpdateService2}, request{&{im.youni.iccs.status.api.StatusUpdateService login0 application/json [0xc82185be60] {false <nil>}}}, response{<nil>}) = err{{"id":"dubbogo.client","code":500,"detail":"Error sending request: dial tcp 116.211.15.189:30880: i/o timeout","status":"Internal Server Error"}}
- 2 把dubbogo/client/rpc_client.go:(rpcClient)call中conn.close和stream.close合并到一个defer clause中，将stream.Close单独交给一个rpcClient.close goroutine处理
- 3 修改dubbogo/registry/zk/client.go & dubbogo/registry/zk/server.go 两个文件中对zk的重连逻辑.

### 2016-08-06 ###
---
- 1 为了让dubbogo/client/rpc_pool.go只保存长连接对象:
    > 给dubbogo/client/rpc_client.go:call 中this.pool.getConn函数调用下面的defer语句段添加StreamingRequest判断条件
- 2 为了检查consumerZookeeperRegistry:Getservice()获取不到service的serviceURL的错误，先给GetService函数和Register函数添加debug log
- 3 把dubbogo/registry/zk/client.go中consumerZookeeperRegistry:services的key由ServiceConfig.String()修改为ServiceConfig.Service，以防止consumerZookeeperRegistry:GetService时检查service是否注册时被提示没有注册的error

### 2016-08-05 ###
---
- 1 dubbogo/selector/cache/cache.go:(cacheSelector)get 修改逻辑，以让超时发生时但无法获取新的serviceURL array的时候继续使用旧的serviceURL array

### 2016-08-03 ###
- 1 把cmg中连接zk的超时时间由5s修改为60s，以进行io timeout测试
     func newZookeeperClient(name string, zkAddrs []string, timeout int)
- 2 给下面三个函数添加serviceURL filter检查
     func (this *consumerZookeeperRegistry) ListServices()
     func (this *consumerZookeeperRegistry) ListService()
     func (this *zookeeperWatcher) watchDir

### 2016-08-02 ###
---
- 1 由于昨天第二项改进后引入了一个新的bug，使得dubbogo/codec/jsonrpc/client.go:(clientCodec)ReadHead中找不到response id对应的method

    > 详细的修改内容见函数dubbogo/codec/jsonrpc/client.go:(clientCodec)Write里面的注释

### 2016-08-02 ###
---
- 1 根据下面panic记录修改 registry/zk/zk_client.go, 确保zookeeperClient{conn}为nil的情况下程序不会panic
    // panic 1
    2016/08/01 11:10:26 Recv loop terminated: err=read tcp 116.211.15.192:59987->116.211.15.190:2181: i/o timeout
    2016/08/01 11:10:26 Send loop terminated: err=<nil>
    [08/01/16 11:10:26] [WARN] get a zookeeper event{zk.Event{Type:-1, State:0, Path:"", Err:error(nil), Server:"116.211.15.190:2181"}}, state{0}:zookeeper disconnected
    [08/01/16 11:10:26] [WARN] zk{addr:[116.211.15.190:2181]} state is StateDisconnected, so close the zk client{name:zk registry}.
    [08/01/16 11:10:26] [INFO] zk{path:[116.211.15.190:2181], name:zk registry} connection goroutine game over.
    2016/08/01 11:10:26 Recv loop terminated: err=read tcp 116.211.15.192:59988->116.211.15.190:2181: i/o timeout
    2016/08/01 11:10:26 Send loop terminated: err=<nil>
    [08/01/16 11:10:26] [WARN] get a zookeeper event{zk.Event{Type:-1, State:0, Path:"", Err:error(nil), Server:"116.211.15.190:2181"}}, state{0}:zookeeper disconnected
    panic: runtime error: invalid memory address or nil pointer dereference
    [signal 0xb code=0x1 addr=0x0 pc=0x888db8]

    goroutine 16 [running]:
    panic(0xa2e9e0, 0xc82000a0e0)
            C:/Program Files (x86)/Console2/go1.6.2/src/runtime/panic.go:481 +0x3e6
    github.com/samuel/go-zookeeper/zk.(*Conn).nextXid(0x0, 0xc800000000)
            C:/Users/AlexStocks/share/test/golang/lib/src/github.com/samuel/go-zookeeper/zk/conn.go:671 +0x58
    github.com/samuel/go-zookeeper/zk.(*Conn).queueRequest(0x0, 0xc, 0x9083c0, 0xc820132d80, 0x908420, 0xc820b3e060, 0xc820177590, 0x0)
            C:/Users/AlexStocks/share/test/golang/lib/src/github.com/samuel/go-zookeeper/zk/conn.go:686 +0x31
    github.com/samuel/go-zookeeper/zk.(*Conn).request(0x0, 0xc80000000c, 0x9083c0, 0xc820132d80, 0x908420, 0xc820b3e060, 0xc820177590, 0x0, 0x0, 0x0)
            C:/Users/AlexStocks/share/test/golang/lib/src/github.com/samuel/go-zookeeper/zk/conn.go:698 +0x92
    :209jjjjjj
    dubbogo/registry/zk.(*zookeeperClient).getChildrenW(0xc82014a000, 0xc8201b0600, 0x34, 0x0, 0x0, 0x0, 0xc820a11730, 0x0, 0x0)
            C:/Users/AlexStocks/share/test/golang/lib/src/dubbogo/registry/zk/zk_client.go:236 +0xc1
    dubbogo/registry/zk.(*zookeeperWatcher).watchDir(0xc82000b710, 0xc8201b0600, 0x34)
            C:/Users/AlexStocks/share/test/golang/lib/src/dubbogo/registry/zk/watch.go:98 +0xe8
    dubbogo/registry/zk.(*zookeeperWatcher).watchService.func2(0xc82000b710, 0xc8201b0600, 0x34)
            C:/Users/AlexStocks/share/test/golang/lib/src/dubbogo/registry/zk/watch.go:215 +0x77
    created by dubbogo/registry/zk.(*zookeeperWatcher).watchService
            C:/Users/AlexStocks/share/test/golang/lib/src/dubbogo/registry/zk/watch.go:217 +0x317

    2016/08/02 14:29:57 Recv loop terminated: err=read tcp 116.211.15.192:39716->116.211.15.190:2181: i/o timeout
    2016/08/02 14:29:57 Send loop terminated: err=<nil>
    panic: runtime error: invalid memory address or nil pointer dereference
    [signal 0xb code=0x1 addr=0x0 pc=0x888db8]

    // panic 2
    goroutine 49 [running]:
    panic(0xa2e9e0, 0xc82000a0e0)
            C:/Program Files (x86)/Console2/go1.6.2/src/runtime/panic.go:481 +0x3e6
    github.com/samuel/go-zookeeper/zk.(*Conn).nextXid(0x0, 0xc800000000)
            C:/Users/AlexStocks/share/test/golang/lib/src/github.com/samuel/go-zookeeper/zk/conn.go:671 +0x58
    github.com/samuel/go-zookeeper/zk.(*Conn).queueRequest(0x0, 0x3, 0x908240, 0xc8201cd2e0, 0x9082a0, 0xc820510cd0, 0xc820ca7c50, 0x0)
            C:/Users/AlexStocks/share/test/golang/lib/src/github.com/samuel/go-zookeeper/zk/conn.go:686 +0x31
    github.com/samuel/go-zookeeper/zk.(*Conn).request(0x0, 0xc800000003, 0x908240, 0xc8201cd2e0, 0x9082a0, 0xc820510cd0, 0xc820ca7c50, 0x0, 0x0, 0x0)
            C:/Users/AlexStocks/share/test/golang/lib/src/github.com/samuel/go-zookeeper/zk/conn.go:698 +0x92
    github.com/samuel/go-zookeeper/zk.(*Conn).ExistsW(0x0, 0xc82006e1a0, 0x19e, 0xc82002d000, 0x0, 0x0, 0x0, 0x0)
            C:/Users/AlexStocks/share/test/golang/lib/src/github.com/samuel/go-zookeeper/zk/conn.go:833 +0x293
    dubbogo/registry/zk.(*zookeeperClient).existW(0xc820016480, 0xc82006e1a0, 0x19e, 0x1, 0x0, 0x0)
            C:/Users/AlexStocks/share/test/golang/lib/src/dubbogo/registry/zk/zk_client.go:289 +0x8d
    dubbogo/registry/zk.(*zookeeperWatcher).watchServiceNode(0xc820504a80, 0xc82006e1a0, 0x19e)
            C:/Users/AlexStocks/share/test/golang/lib/src/dubbogo/registry/zk/watch.go:59 +0xb9
    dubbogo/registry/zk.(*zookeeperWatcher).watchService.func1(0xc820504a80, 0xc820504ae0, 0xc82006e1a0, 0x19e, 0xc8200a41b0)
            C:/Users/AlexStocks/share/test/golang/lib/src/dubbogo/registry/zk/watch.go:205 +0x4b
    created by dubbogo/registry/zk.(*zookeeperWatcher).watchService
            C:/Users/AlexStocks/share/test/golang/lib/src/dubbogo/registry/zk/watch.go:209 +0x1016

- 2 jsonrpc java端程序不能解析jsonrpc 2.0中id超过uint32_max的数字, 在dubbogo/codec/jsonrpc/client.go对Id进行mask(MAX_JSONRPC_ID)运算，防止其超过int32_max
     return error{"code":-32603,"message":"json: cannot unmarshal number -1874143215 into Go value of type uint64"}
- 3 http抓包方法 tcpdump  -XvvennSs 0 -i any tcp[20:2]=0x4745 or tcp[20:2]=0x4854
     tcpdump -Xeennvvlps0 port 30880 -iany
     sar 1 0 -n DEV -e 15:00:00 > data.txt
     //每隔1秒记录网络使用情况，直到15点，数据将保存到data.txt文件中

### 2016-07-29 ###
---
- 1 为dubbogo/client/rpc_client.go:rpcClient添加了Id field
- 2 为serverCodec:ReadHeader添加请求方法是否是POST验证条件
- 3 根据下面两个命令追踪的http response流程，对httpTransportSocket:Send进行了优化
     tcpdump -Xnlpvvs0 -S port 10000 -iany -w http.pcap
     strace -p 32732 -f

### 2016-07-28 ###
---
- 1 参照effecive go的leaky buffer的例子，修改dubbogo/server/rpc_server.go的server:

     type server struct {
         mu         sync.Mutex          // protects the serviceMap
         serviceMap map[string]*service // service name -> service
         // reqLock    sync.Mutex          // protects freeReq
         // freeReq    *request
         freeReq chan *request
         // respLock sync.Mutex // protects freeRsp
         // freeRsp *response
         freeRsp  chan *response
         listener transport.Listener
     }

     把lock和list修改为一个有size限制的req/rsp chan，通过select把锁也省略掉.

- 2 参考net/netutil/listen.go进行最大连接数限制
     type httpTransportListener {
         sem chan struct{}
     }
- 3 给dubbogo/client & dubbogo/server添加once，以确保安全退出

### dubbogo中的默认值 ###
---
- 1 dubbogo/selector/cache/cache.go
    var (
    // selector每15分钟通过tick函数清空cache或者get函数去清空某个service的cache，
        // 以全量获取某个service的所有providers
        DefaultTTL = 5 * time.Minute
    )

- 2 dubbogo/server/http_transport.go
     DefaultMaxSleepTime            = 1 * time.Second  // accept中间最大sleep interval
     DefaultMAXConnNum              = 50 * 1024 * 1024 // 默认最大连接数 50w

- 3 dubbogo/server/rpc_server.go
    const (
        FREE_LIST_SIZE = 4 * 1024 // freeReq & freeRsp leaky buffer size
    )

- 4 dubbogo/registry/zk/watch.go
    const (
    ZK_DELAY                    = 3  // watchDir中使用，防止不断地对zk重连
    MAX_TIMES               = 10 // 设置(wathcer)watchDir()等待时长
    Wactch_Event_Channel_Size   = 32 // 用于设置通知selector的event channel的size
    ZKCLIENT_EVENT_CHANNEL_SIZE = 4  // 设置用于zk client与watcher&consumer&provider之间沟通的channel的size
    )