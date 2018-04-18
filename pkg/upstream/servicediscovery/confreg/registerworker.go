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
)

const (
    TASK_TYPE_PUBLISH   = 0
    TASK_TYPE_SUBSCRIBE = 1
)

type registryStreamContext struct {
    streamId         string
    registryRequest  interface{}
    registryResponse *model.RegisterResponsePb
    finished         chan bool
    err              error
}

type registryTask struct {
    taskId      string
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

    publisherHolder         sync.Map
    subscriberHolder        sync.Map
    registryTaskQueue       *list.List
    registryFailedTaskQueue *list.List

    initialized      bool
    stopChan         chan bool
    registryWorkChan chan bool
}

func NewRegisterWorker(sysConfig *config.SystemConfig, registryConfig *config.RegistryConfig,
    confregServerManager *servermanager.RegistryServerManager, rpcServerManager servermanager.RPCServerManager) *registerWorker {
    rw := &registerWorker{
        systemConfig:            sysConfig,
        registryConfig:          registryConfig,
        confregServerManager:    confregServerManager,
        rpcServerManager:        rpcServerManager,
        registryTaskQueue:       list.New(),
        registryFailedTaskQueue: list.New(),
        initialized:             false,
        stopChan:                make(chan bool, 1),
        registryWorkChan:        make(chan bool),
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
        panic("Can not connect to confreg server. Because confreg server list is empty.")
    }
    rw.connectedConfregServer = confregServer
    rw.newCodecClient(confregServer)
}

func (rw *registerWorker) newCodecClient(confregServer string) {
    remoteAddr, _ := net.ResolveTCPAddr("tcp", confregServer)
    conn := network.NewClientConnection(nil, remoteAddr, rw.stopChan)
    receiveDataListener := NewReceiveDataListener(rw.rpcServerManager)
    codecClient := stream.NewBiDirectCodeClient(protocol.SofaRpc, conn, nil, receiveDataListener)
    conn.Connect(true)
    rw.codecClient = &codecClient
    log.DefaultLogger.Infof("Connect to confreg server. server = %v", confregServer)
}

func (rw *registerWorker) SubmitPublishTask(dataId string, data []string, eventType string) {
    publisher := rw.getPublisher(dataId)

    registryTask := registryTask{
        taskType:    TASK_TYPE_PUBLISH,
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
        taskType:    TASK_TYPE_SUBSCRIBE,
        eventType:   eventType,
        subscriber:  subscriber,
        sendCounter: metrics.NewCounter(),
    }

    rw.registryTaskQueue.PushBack(registryTask)

    rw.registryWorkChan <- true
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
    registryTask := ele.Value.(registryTask)
    registryTask.taskId = RandomUuid()

    var err error
    //Retry 2 times
    for i := 0; i < 2; i++ {
        time.Sleep(CalRetreatTime(registryTask.sendCounter.Count(), 5))

        if registryTask.taskType == TASK_TYPE_PUBLISH {
            publisher := registryTask.publisher
            err = publisher.doWork(registryTask.taskId, registryTask.data, registryTask.eventType)
            publisher.version++
        } else {
            sub := registryTask.subscriber
            err = sub.doWork(registryTask.taskId, registryTask.eventType)
            sub.version++
        }

        if err == nil {
            registryTask.sendCounter.Clear()
            return nil
        } else {
            registryTask.sendCounter.Inc(1)
            log.DefaultLogger.Errorf("Register failed at %d times. Task = %v, error = %v", registryTask.sendCounter.Count(),
                registryTask, err)
        }
    }
    return errors.New("register failed")

}

func (rw *registerWorker) PublishSync(dataId string, data []string) error {
    return rw.getPublisher(dataId).doWork(RandomUuid(), data, model.EventTypePb_REGISTER.String())
}

func (rw *registerWorker) UnPublishSync(dataId string, data []string) error {
    return rw.getPublisher(dataId).doWork(RandomUuid(), data, model.EventTypePb_UNREGISTER.String())
}

func (rw *registerWorker) SubscribeSync(dataId string) error {
    return rw.getSubscriber(dataId).doWork(RandomUuid(), model.EventTypePb_REGISTER.String())
}

func (rw *registerWorker) UnSubscribeSync(dataId string) error {
    return rw.getSubscriber(dataId).doWork(RandomUuid(), model.EventTypePb_UNREGISTER.String())
}

func (rw *registerWorker) getPublisher(dataId string) *Publisher {
    storedPublisher, ok := rw.publisherHolder.Load(dataId)
    if ok {
        return storedPublisher.(*Publisher)
    }
    newPublisher := NewPublisher(dataId, rw.codecClient, rw.registryConfig, rw.systemConfig)
    rw.publisherHolder.Store(dataId, newPublisher)
    return newPublisher
}

func (rw *registerWorker) getSubscriber(dataId string) *Subscriber {
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
        t := time.NewTimer(rw.registryConfig.ScheduleRegisterTaskDuration)
        <-t.C
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
    confregServer, _ := rw.confregServerManager.GetRegistryServerByRandom()
    rw.newCodecClient(confregServer)
    rw.connectedConfregServer = confregServer
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
