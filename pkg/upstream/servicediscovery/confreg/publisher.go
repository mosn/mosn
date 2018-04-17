package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/config"
    "time"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
    "gitlab.alipay-inc.com/afe/mosn/pkg/stream"
    "github.com/golang/protobuf/proto"
    "gitlab.alipay-inc.com/afe/mosn/pkg/types"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "gitlab.alipay-inc.com/afe/mosn/pkg/network/buffer"
    "fmt"
    "errors"
)

type Publisher struct {
    registryConfig *config.RegistryConfig
    sysConfig      *config.SystemConfig
    codecClient    *stream.CodecClient

    registerId string
    dataId     string

    version       int64
    startTime     int32
    streamContext *registryStreamContext
}

func NewPublisher(dataId string, codecClient *stream.CodecClient, registryConfig *config.RegistryConfig,
    systemConfig *config.SystemConfig) *Publisher {
    p := &Publisher{
        dataId:         dataId,
        version:        0,
        registryConfig: registryConfig,
        sysConfig:      systemConfig,
        codecClient:    codecClient,
    }

    return p
}

//sync call. not thread safe
func (p *Publisher) doWork(taskId string, svrHost []string, eventType string) error {
    defer func() {
        if err := recover(); err != nil {
            log.DefaultLogger.Errorf("Publish data to confreg failed. eventType = %s, "+
                "dataId = %s, registerId = %s, server = %v. error = %v",
                eventType, p.dataId, p.registerId, svrHost, err)
        }
    }()
    //1. Assemble request
    p.registerId = taskId
    request := p.assemblePublisherRegisterPb(svrHost, eventType)
    body, _ := proto.Marshal(request)
    //2. Send request
    p.sendRequest(taskId, body)
    //3. Handle response
    p.streamContext = &registryStreamContext{
        streamId:        taskId,
        registryRequest: request,
        finished:        make(chan bool),
    }

    return p.handleResponse(taskId, request)
}

func (p *Publisher) sendRequest(taskId string, body []byte) {
    streamEncoder := (*p.codecClient).NewStream(taskId, p)
    headers := BuildBoltPublishRequestCommand(len(body))
    streamEncoder.EncodeHeaders(headers, false)
    streamEncoder.EncodeData(buffer.NewIoBufferBytes(body), true)
}

func (p *Publisher) handleResponse(taskId string, request *model.PublisherRegisterPb) error {
    t := time.NewTimer(p.registryConfig.RegisterTimeout)
    for ; ; {
        select {
        case <-t.C:
            {
                errMsg := fmt.Sprintf("Publish data to confreg timeout. register id = %v", taskId)
                log.DefaultLogger.Errorf(errMsg)
                return errors.New(errMsg)
            }
        case <-p.streamContext.finished:
            {
                pubResponse := p.streamContext.registryResponse

                if p.streamContext.err == nil && pubResponse.Success && !pubResponse.Refused {
                    log.DefaultLogger.Infof("Publish data to confreg success. register id = %v", pubResponse.RegistId)
                    return nil
                }
                errMsg := fmt.Sprintf("Publish data to confreg failed. register id = %v, message = %v",
                    pubResponse.RegistId, pubResponse.Message)
                log.DefaultLogger.Errorf(errMsg)
                return errors.New(errMsg)
            }
        }
    }
}

func (p *Publisher) OnDecodeData(data types.IoBuffer, endStream bool) {
    if !endStream {
        return
    }
    defer func() {
        p.streamContext.finished <- true
        data.Reset()
    }()

    responseStream := data.Bytes()
    response := &model.RegisterResponsePb{}

    if err := proto.Unmarshal(responseStream, response); err != nil {
        p.streamContext.err = err
        log.DefaultLogger.Errorf("Unmarshal registry result failed. error = %v", err)
        return
    }

    p.streamContext.registryResponse = response
}

func (p *Publisher) OnDecodeTrailers(trailers map[string]string) {
}

func (p *Publisher) OnDecodeHeaders(headers map[string]string, endStream bool) {
    if !endStream {
        return
    }
    //检查taskId是否是当前，如果不是就抛弃
    fmt.Println(headers)
}

func (p *Publisher) assemblePublisherRegisterPb(svrHost []string, eventType string) *model.PublisherRegisterPb {
    br := &model.BaseRegisterPb{
        InstanceId: p.sysConfig.InstanceId,
        Zone:       p.sysConfig.Zone,
        AppName:    p.sysConfig.AppName,
        DataId:     p.dataId,
        Group:      CONFREG_SOFA_GROUP,
        ProcessId:  RandomUuid(),
        RegistId:   p.registerId,
        ClientId:   p.registerId,
        EventType:  eventType,
        Version:    p.version,
        Timestamp:  time.Now().Unix(),
    }

    var dataList []*model.DataBoxPb
    dataLen := len(svrHost)
    if dataLen == 0 {
        dataList = nil
    } else {
        dataList = make([]*model.DataBoxPb, dataLen)
        for i, data := range svrHost {
            dataList[i] = &model.DataBoxPb{Data: data}
        }
    }

    pr := &model.PublisherRegisterPb{
        DataList:     dataList,
        BaseRegister: br,
    }

    p.version = p.version + 1

    return pr
}
