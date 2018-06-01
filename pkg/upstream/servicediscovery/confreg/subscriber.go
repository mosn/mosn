package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/stream"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
    //"strconv"
    //"sync/atomic"
    "math/rand"
    
    "time"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "github.com/golang/protobuf/proto"
    "gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
    "gitlab.alipay-inc.com/afe/mosn/pkg/types"
    "fmt"
    "errors"
    "sync"
    "gitlab.alipay-inc.com/afe/mosn/pkg/protocol/sofarpc"
)

var subLock = new(sync.Mutex)

type Subscriber struct {
    systemConfig        *config.SystemConfig
    registryConfig      *config.RegistryConfig
    codecClient         *stream.CodecClient
    registerId          string
    dataId              string
    scope               string
    version             int64
    lastActionEventType string
    streamContext       *registryStreamContext
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
        version:        1,
    }

    return sub
}

func (s *Subscriber) redo() error {
    if s.lastActionEventType == model.EventTypePb_REGISTER.String() {
        log.DefaultLogger.Infof("Resubscribe data. data id = %s", s.dataId)
        s.doWork(s.lastActionEventType)
    }
    return nil
}

func (s *Subscriber) doWork(eventType string) error {
    subLock.Lock()

    s.lastActionEventType = eventType

    defer func() {
        subLock.Unlock()
    }()

    //1. Assemble request
    request := s.assembleSubscriberRegisterPb(eventType)
    body, _ := proto.Marshal(request)
    
    //2. Send request
    reqIdstr := fmt.Sprintf("%d", rand.Uint32())
    err := s.sendRequest(reqIdstr, body)
    if err != nil {
        return err
    }
    s.streamContext = &registryStreamContext{
        streamId:        reqIdstr,
        registryRequest: request,
        finished:        make(chan bool),
        mismatch:        false,
    }
    //3. Handle response
    return s.handleResponse(request)
}

func (s *Subscriber) sendRequest(reqId string, body []byte) error {
    streamEncoder := (*s.codecClient).NewStream(reqId, s)
    headers := BuildBoltSubscribeRequestCommand(len(body), reqId)
    err := streamEncoder.EncodeHeaders(headers, false)
    if err != nil {
        return err
    }
    return streamEncoder.EncodeData(buffer.NewIoBufferBytes(body), true)
}

func (s *Subscriber) handleResponse(request *model.SubscriberRegisterPb) error {
    for ; ; {
        select {
        case <-time.After(s.registryConfig.RegisterTimeout):
            {
                errMsg := fmt.Sprintf("Subscribe data from confreg timeout.  data id = %s, register id = %v",
                    s.dataId, s.registerId)
                log.DefaultLogger.Errorf(errMsg)
                return errors.New(errMsg)
            }
        case <-s.streamContext.finished:
            {
                subResponse := s.streamContext.registryResponse

                if s.streamContext.err == nil && subResponse.Success && !subResponse.Refused {
                    log.DefaultLogger.Infof("Subscribe data from confreg success. data id = %s, register id = %v",
                        s.dataId, subResponse.RegistId)
                    return nil
                }
                errMsg := fmt.Sprintf("Subscribe data from confreg failed.  data id = %s, register id = %v, message = %v",
                    s.dataId, subResponse.RegistId, subResponse.Message)
                log.DefaultLogger.Errorf(errMsg)
                return errors.New(errMsg)
            }
        }
    }
}

func (s *Subscriber) OnDecodeHeaders(headers map[string]string, endStream bool) {
    boltReqId := headers[sofarpc.HeaderReqID]
    if boltReqId != s.streamContext.streamId {
        errMsg := fmt.Sprintf("Received mismatch subscribe response. data id = %s, received reqId = %s, context reqId = %s",
            s.dataId, boltReqId, s.streamContext.streamId)
        s.streamContext.mismatch = true
        log.DefaultLogger.Errorf(errMsg)
    }
}

func (s *Subscriber) OnDecodeData(data types.IoBuffer, endStream bool) {
    if !endStream {
        return
    }
    if s.streamContext.mismatch {
        s.streamContext.mismatch = false
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
        log.DefaultLogger.Errorf("Unmarshal registry result failed. data id = %s, error = %v", s.dataId, err)
        return
    }

    s.streamContext.registryResponse = response
}

func (s *Subscriber) OnDecodeTrailers(trailers map[string]string) {

}

func (p *Subscriber) OnDecodeError(err error,headers map[string]string){

}

func (s *Subscriber) assembleSubscriberRegisterPb(eventType string) *model.SubscriberRegisterPb {
    br := &model.BaseRegisterPb{
        InstanceId: s.systemConfig.InstanceId,
        Zone:       s.systemConfig.Zone,
        AppName:    s.systemConfig.AppName,
        DataId:     appendDataIdSuffix(s.dataId),
        Group:      ConfregSofaGroup,
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

    s.version = s.version + 1
    return sr
}
