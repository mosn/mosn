package registry

import (
    "time"
    "container/list"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "sync"
    "gitlab.alipay-inc.com/afe/mosn/pkg/stream"
    "net"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "gitlab.alipay-inc.com/afe/mosn/pkg/network"
    "gitlab.alipay-inc.com/afe/mosn/pkg/protocol"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "github.com/rcrowley/go-metrics"
    "errors"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
    "gitlab.alipay-inc.com/afe/mosn/pkg/types"
)

const (
    PublishTask    = 0
    SubscriberTask = 1
)

type registryStreamContext struct {
    streamId         string
    registryRequest  interface{}
    registryResponse *model.RegisterResponsePb
    finished         chan bool
    err              error
    mismatch         bool
}

type registryTask struct {
    dataId      string
    taskType    int
    eventType   string
    subscriber  *Subscriber
    publisher   *Publisher
    data        []string
    sendCounter metrics.Counter
}

type registerWorker struct {
    systemConfig           *config.SystemConfig
    registryConfig         *config.RegistryConfig
    confregServerManager   *servermanager.RegistryServerManager
    rpcServerManager       servermanager.RPCServerManager
    codecClient            *stream.CodecClient
    connectedConfregServer string

    publisherHolder   sync.Map
    subscriberHolder  sync.Map
    registryTaskQueue *list.List
    pubMutex          *sync.Mutex
    subMutex          *sync.Mutex

    initialized      bool
    stopChan         chan bool
    registryWorkChan chan bool
}

func NewRegisterWorker(sysConfig *config.SystemConfig, registryConfig *config.RegistryConfig,
    confregServerManager *servermanager.RegistryServerManager, rpcServerManager servermanager.RPCServerManager) *registerWorker {
    rw := &registerWorker{
        systemConfig:         sysConfig,
        registryConfig:       registryConfig,
        confregServerManager: confregServerManager,
        rpcServerManager:     rpcServerManager,
        registryTaskQueue:    list.New(),
        pubMutex:             new(sync.Mutex),
        subMutex:             new(sync.Mutex),
        initialized:          false,
        stopChan:             make(chan bool, 1),
        registryWorkChan:     make(chan bool),
    }

    rw.init()

    confregServerManager.RegisterServerChangeListener(rw)

    go rw.work()
    go rw.scheduleWorkAtFixTime()

    return rw
}

func (rw *registerWorker) init() {
    if rw.initialized {
        return
    }
    rw.initialized = true

    confregServer, ok := rw.confregServerManager.GetRegistryServerByRandom()
    if !ok {
        errMsg := "can not connect to confreg server. Because confreg server list is empty"
        log.DefaultLogger.Errorf(errMsg)
        panic(errMsg)
    }
    for err := rw.newCodecClient(confregServer); err != nil; {
        log.DefaultLogger.Warnf("Connect to confreg server failed. error = %v", err)
        time.Sleep(rw.registryConfig.ConnectRetryDuration)
        confregServer, ok = rw.confregServerManager.GetRegistryServerByRR()
        err = rw.newCodecClient(confregServer)
    }
}

func (rw *registerWorker) newCodecClient(confregServer string) error {
    rw.connectedConfregServer = confregServer

    remoteAddr, _ := net.ResolveTCPAddr("tcp", confregServer)
    conn := network.NewClientConnection(nil, remoteAddr, rw.stopChan, log.DefaultLogger)
    receiveDataListener := NewReceiveDataListener(rw.rpcServerManager)
    codecClient := stream.NewBiDirectCodeClient(nil, protocol.SofaRpc, conn, nil, receiveDataListener)
    codecClient.AddConnectionCallbacks(rw)

    err := conn.Connect(true)
    if err != nil {
        return err
    }

    rw.codecClient = &codecClient
    log.DefaultLogger.Infof("Connect to confreg server. server = %v", confregServer)
    return nil
}

func (rw *registerWorker) SubmitPublishTask(dataId string, data []string, eventType string) {
    publisher := rw.getPublisher(dataId)

    registryTask := registryTask{
        dataId:      dataId,
        taskType:    PublishTask,
        eventType:   eventType,
        publisher:   publisher,
        data:        data,
        sendCounter: metrics.NewCounter(),
    }
    rw.registryTaskQueue.PushBack(registryTask)

    rw.registryWorkChan <- true
}

func (rw *registerWorker) SubmitSubscribeTask(dataId string, eventType string) {
    subscriber := rw.getSubscriber(dataId)

    registryTask := registryTask{
        dataId:      dataId,
        taskType:    SubscriberTask,
        eventType:   eventType,
        subscriber:  subscriber,
        sendCounter: metrics.NewCounter(),
    }

    rw.registryTaskQueue.PushBack(registryTask)

    rw.registryWorkChan <- true
}

func (rw *registerWorker) PublishSync(dataId string, data []string) error {
    return rw.getPublisher(dataId).doWork(data, model.EventTypePb_REGISTER.String())
}

func (rw *registerWorker) UnPublishSync(dataId string, data []string) error {
    return rw.getPublisher(dataId).doWork(data, model.EventTypePb_UNREGISTER.String())
}

func (rw *registerWorker) SubscribeSync(dataId string) error {
    return rw.getSubscriber(dataId).doWork(model.EventTypePb_REGISTER.String())
}

func (rw *registerWorker) UnSubscribeSync(dataId string) error {
    return rw.getSubscriber(dataId).doWork(model.EventTypePb_UNREGISTER.String())
}

func (rw *registerWorker) work() {
    for ; ; {
        select {
        case <-rw.registryWorkChan:
            {
                log.DefaultLogger.Infof("Start consume registry task. task queue size = %d", rw.registryTaskQueue.Len())
                var next *list.Element
                for ele := rw.registryTaskQueue.Front(); ele != nil; {
                    next = ele.Next()
                    if err := rw.doRegister(ele); err == nil {
                        rw.registryTaskQueue.Remove(ele)
                    }
                    ele = next
                }
            }
        case <-rw.stopChan:
            {
                break
            }
        }
    }
}

func (rw *registerWorker) doRegister(ele *list.Element) error {
    defer func() {
        //Keeping register work thread alive
        if err := recover(); err != nil {
            log.DefaultLogger.Fatal("Some unexpected exception occurred. err = ", err)
        }
    }()

    registryTask := ele.Value.(registryTask)
    var err error
    //Retry 2 times
    for i := 0; i < 2; i++ {
        time.Sleep(CalRetreatTime(registryTask.sendCounter.Count(), 5))

        if registryTask.taskType == PublishTask {
            publisher := registryTask.publisher
            err = publisher.doWork(registryTask.data, registryTask.eventType)
        } else {
            sub := registryTask.subscriber
            err = sub.doWork(registryTask.eventType)
        }

        if err == nil {
            registryTask.sendCounter.Clear()
            return nil
        } else {
            registryTask.sendCounter.Inc(1)
            log.DefaultLogger.Errorf("Register failed at %d times. data id = %v, error = %v", registryTask.sendCounter.Count(),
                registryTask.dataId, err)
        }
    }
    return errors.New("register failed")

}

func (rw *registerWorker) getPublisher(dataId string) *Publisher {
    rw.pubMutex.Lock()
    defer func() {
        rw.pubMutex.Unlock()
    }()
    storedPublisher, ok := rw.publisherHolder.Load(dataId)
    if ok {
        return storedPublisher.(*Publisher)
    }
    newPublisher := NewPublisher(dataId, rw.codecClient, rw.registryConfig, rw.systemConfig)
    rw.publisherHolder.Store(dataId, newPublisher)
    return newPublisher
}

func (rw *registerWorker) getSubscriber(dataId string) *Subscriber {
    rw.subMutex.Lock()
    defer func() {
        rw.subMutex.Unlock()
    }()
    storedSubscriber, ok := rw.subscriberHolder.Load(dataId)
    if ok {
        return storedSubscriber.(*Subscriber)
    }
    newSubscriber := NewSubscriber(dataId, rw.codecClient, rw.registryConfig, rw.systemConfig)
    rw.subscriberHolder.Store(dataId, newSubscriber)
    return newSubscriber
}

func (rw *registerWorker) scheduleWorkAtFixTime() {
    for ; ; {
        <-time.After(rw.registryConfig.ScheduleCompensateRegisterTaskDuration)
        rw.registryWorkChan <- true
    }
}

func (rw *registerWorker) OnRegistryServerChangeEvent(registryServers []string) {
    var isConnectedServerStillAlive = false
    for _, s := range registryServers {
        if s == rw.connectedConfregServer {
            isConnectedServerStillAlive = true
            break
        }
    }
    if isConnectedServerStillAlive {
        return
    }
    rw.refreshCodecClient()
}

func (rw *registerWorker) OnEvent(event types.ConnectionEvent) {
    if !event.IsClose() {
        return
    }
    log.DefaultLogger.Warnf("The Connection with confreg server closed, will connect to another server. "+
        "current connected confreg server = %s, event type = %v", rw.connectedConfregServer, event)

    rw.refreshCodecClient()

    rw.publisherHolder.Range(rePub)
    rw.subscriberHolder.Range(reSub)
}

func (rw *registerWorker) refreshCodecClient() {
    for {
        confregServer, ok := rw.confregServerManager.GetRegistryServerByRR()
        if !ok {
            log.DefaultLogger.Warnf("Confreg server list is empty.")
            time.Sleep(rw.registryConfig.ConnectRetryDuration)
            continue
        }
        err := rw.newCodecClient(confregServer)
        if err != nil {
            log.DefaultLogger.Warnf("Connect to confreg server failed. error = %v", err)
            time.Sleep(rw.registryConfig.ConnectRetryDuration)
            continue
        }
        break
    }
    rw.publisherHolder.Range(rw.refreshPublisherCodecClient)
    rw.subscriberHolder.Range(rw.refreshSubscriberCodecClient)
}

func (rw *registerWorker) refreshPublisherCodecClient(key, value interface{}) bool {
    value.(*Publisher).codecClient = rw.codecClient
    return true
}

func (rw *registerWorker) refreshSubscriberCodecClient(key, value interface{}) bool {
    value.(*Subscriber).codecClient = rw.codecClient
    return true
}

func rePub(key, value interface{}) bool {
    if err := value.(*Publisher).redo(); err != nil {
        return false
    }
    return true
}

func reSub(key, value interface{}) bool {
    if err := value.(*Subscriber).redo(); err != nil {
        return false
    }
    return true
}

func (rw *registerWorker) OnAboveWriteBufferHighWatermark() {
}

func (rw *registerWorker) OnBelowWriteBufferLowWatermark() {
}
