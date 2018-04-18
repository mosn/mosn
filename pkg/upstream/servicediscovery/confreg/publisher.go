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
    "math/rand"
    "sync"
)

var pubLock = new(sync.Mutex)

type Publisher struct {
    registryConfig *config.RegistryConfig
    sysConfig      *config.SystemConfig
    codecClient    *stream.CodecClient
    registerId     string
    dataId         string
    version        int64
    startTime      int32
    streamContext  *registryStreamContext
}

func NewPublisher(dataId string, codecClient *stream.CodecClient, registryConfig *config.RegistryConfig,
    systemConfig *config.SystemConfig) *Publisher {
    p := &Publisher{
        version:        1,
        dataId:         dataId,
        registryConfig: registryConfig,
        sysConfig:      systemConfig,
        codecClient:    codecClient,
        registerId:     RandomUuid(),
    }

    return p
}

//sync call. not thread safe
func (p *Publisher) doWork(svrHost []string, eventType string) error {
    pubLock.Lock()
    defer func() {
        pubLock.Unlock()
    }()
    //1. Assemble request
    request := p.assemblePublisherRegisterPb(svrHost, eventType)
    body, _ := proto.Marshal(request)
    //2. Send request
    streamId := fmt.Sprintf("%d", rand.Uint32())
    err := p.sendRequest(streamId, body)
    if err != nil {
        return err
    }
    //3. Handle response
    p.streamContext = &registryStreamContext{
        streamId:        streamId,
        registryRequest: request,
        finished:        make(chan bool),
        mismatch:        false,
    }

    return p.handleResponse(request)
}

func (p *Publisher) sendRequest(streamId string, body []byte) error {
    streamEncoder := (*p.codecClient).NewStream(streamId, p)
    headers := BuildBoltPublishRequestCommand(len(body), streamId)
    err := streamEncoder.EncodeHeaders(headers, false)
    if err != nil {
        return err
    }
    return streamEncoder.EncodeData(buffer.NewIoBufferBytes(body), true)
}

func (p *Publisher) handleResponse(request *model.PublisherRegisterPb) error {
    for ; ; {
        select {
        case <-time.After(p.registryConfig.RegisterTimeout):
            {
                errMsg := fmt.Sprintf("Publish data to confreg timeout. data id = %s, register id = %v", p.dataId, p.registerId)
                log.DefaultLogger.Errorf(errMsg)
                return errors.New(errMsg)
            }
        case <-p.streamContext.finished:
            {
                pubResponse := p.streamContext.registryResponse

                if p.streamContext.err == nil && pubResponse.Success && !pubResponse.Refused {
                    log.DefaultLogger.Infof("Publish data to confreg success. data id = %v, register id = %v",
                        p.dataId, pubResponse.RegistId)
                    return nil
                }

                errMsg := fmt.Sprintf("Publish data to confreg failed.  data id = %s, register id = %v, message = %v",
                    p.dataId, pubResponse.RegistId, pubResponse.Message)
                log.DefaultLogger.Errorf(errMsg)
                return errors.New(errMsg)
            }
        }
    }
}

func (p *Publisher) OnDecodeHeaders(headers map[string]string, endStream bool) {
    boltReqId := headers["x-mosn-sofarpc-headers-property-requestid"]
    if boltReqId != p.streamContext.streamId {
        errMsg := fmt.Sprintf("Received mismatch subscribe response. data id = %s, received reqId = %s, context reqId = %s",
            p.dataId, boltReqId, p.streamContext.streamId)
        p.streamContext.mismatch = true
        log.DefaultLogger.Errorf(errMsg)
    }
}

func (p *Publisher) OnDecodeData(data types.IoBuffer, endStream bool) {
    if !endStream {
        return
    }
    //Ignore mismatch response.
    if p.streamContext.mismatch {
        p.streamContext.mismatch = false
        return
    }

    defer func() {
        p.streamContext.finished <- true
        data.Reset()
    }()

    if p.streamContext.err != nil {
        return
    }

    responseStream := data.Bytes()
    response := &model.RegisterResponsePb{}

    if err := proto.Unmarshal(responseStream, response); err != nil {
        p.streamContext.err = err
        log.DefaultLogger.Errorf("Unmarshal registry result failed. data id = %s, error = %v", p.dataId, err)
        return
    }

    p.streamContext.registryResponse = response
}

func (p *Publisher) OnDecodeTrailers(trailers map[string]string) {
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
