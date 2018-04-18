package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/stream"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
    "time"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "github.com/golang/protobuf/proto"
    "gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
    "gitlab.alipay-inc.com/afe/mosn/pkg/types"
    "fmt"
    "errors"
    "math/rand"
)

type Subscriber struct {
    systemConfig   *config.SystemConfig
    registryConfig *config.RegistryConfig
    codecClient    *stream.CodecClient
    registerId     string
    dataId         string
    scope          string
    version        int64
    streamContext  *registryStreamContext
}

func NewSubscriber(dataId string, client *stream.CodecClient,
    registryConfig *config.RegistryConfig, systemConfig *config.SystemConfig) *Subscriber {

    sub := &Subscriber{
        systemConfig:   systemConfig,
        registryConfig: registryConfig,
        codecClient:    client,
        registerId:     RandomUuid(),
        dataId:         dataId,
        scope:          "global",
        version:        0,
    }

    return sub
}

func (s *Subscriber) doWork(taskId string, eventType string) error {
    defer func() {
        if err := recover(); err != nil {
            log.DefaultLogger.Errorf("Registry to confreg failed. eventType = %s, dataId = %s, registerId = %s error = %v.",
                eventType, s.dataId, s.registerId, err)
        }
    }()
    //1. Assemble request
    s.registerId = taskId
    request := s.assembleSubscriberRegisterPb(eventType)
    body, _ := proto.Marshal(request)
    //2. Send request
    reqId := fmt.Sprintf("%d", rand.Uint32())
    err := s.sendRequest(taskId, reqId, body)
    if err != nil {
        return err
    }
    s.streamContext = &registryStreamContext{
        streamId:        reqId,
        registryRequest: request,
        finished:        make(chan bool),
    }
    //3. Handle response
    return s.handleResponse(taskId, request)
}

func (s *Subscriber) sendRequest(taskId string, reqId string, body []byte) error {
    streamEncoder := (*s.codecClient).NewStream(taskId, s)
    headers := BuildBoltSubscribeRequestCommand(len(body), reqId)
    err := streamEncoder.EncodeHeaders(headers, false)
    if err != nil {
        return err
    }
    return streamEncoder.EncodeData(buffer.NewIoBufferBytes(body), true)
}

func (s *Subscriber) handleResponse(taskId string, request *model.SubscriberRegisterPb) error {
    t := time.NewTimer(s.registryConfig.RegisterTimeout)
    for ; ; {
        select {
        case <-t.C:
            {
                errMsg := fmt.Sprintf("Subscribe data from confreg timeout. register id = %v", taskId)
                log.DefaultLogger.Errorf(errMsg)
                return errors.New(errMsg)
            }
        case <-s.streamContext.finished:
            {
                subResponse := s.streamContext.registryResponse

                if s.streamContext.err == nil && subResponse.Success && !subResponse.Refused {
                    log.DefaultLogger.Infof("Subscribe data from confreg success. register id = %v", subResponse.RegistId)
                    return nil
                }
                errMsg := fmt.Sprintf("Subscribe data from confreg failed. register id = %v, message = %v",
                    subResponse.RegistId, subResponse.Message)
                log.DefaultLogger.Errorf(errMsg)
                return errors.New(errMsg)
            }
        }
    }
}

func (s *Subscriber) OnDecodeData(data types.IoBuffer, endStream bool) {
    if !endStream {
        return
    }

    defer func() {
        s.streamContext.finished <- true
        data.Reset()
    }()

    if s.streamContext.err != nil {
        return
    }

    responseStream := data.Bytes()
    response := &model.RegisterResponsePb{}

    if err := proto.Unmarshal(responseStream, response); err != nil {
        s.streamContext.err = err
        log.DefaultLogger.Errorf("Unmarshal registry result failed. error = %v", err)
        return
    }

    s.streamContext.registryResponse = response
}

func (s *Subscriber) OnDecodeTrailers(trailers map[string]string) {

}

func (s *Subscriber) OnDecodeHeaders(headers map[string]string, endStream bool) {
    if !endStream {
        return 
    }
    boltReqId := headers["x-mosn-sofarpc-headers-property-requestid"]
    if boltReqId != s.streamContext.streamId {
        errMsg := fmt.Sprintf("Received mismatch subscribe response. received reqId = %s, context reqId = %s",
            boltReqId, s.streamContext.streamId)
        s.streamContext.err = errors.New(errMsg)
        log.DefaultLogger.Errorf(errMsg)
    }
}

func (s *Subscriber) assembleSubscriberRegisterPb(eventType string) *model.SubscriberRegisterPb {
    br := &model.BaseRegisterPb{
        InstanceId: s.systemConfig.InstanceId,
        Zone:       s.systemConfig.Zone,
        AppName:    s.systemConfig.AppName,
        DataId:     s.dataId,
        Group:      CONFREG_SOFA_GROUP,
        ProcessId:  RandomUuid(),
        RegistId:   s.registerId,
        ClientId:   s.registerId,
        EventType:  eventType,
        Version:    s.version,
        Timestamp:  time.Now().Unix(),
    }

    sr := &model.SubscriberRegisterPb{
        Scope:        s.scope,
        BaseRegister: br,
    }

    return sr
}
