package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/types"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
    "github.com/golang/protobuf/proto"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
)

type receiveDataListener struct {
    rpcServerManager servermanager.RPCServerManager
}

func NewReceiveDataListener(rpcServerManager servermanager.RPCServerManager) *receiveDataListener {
    return &receiveDataListener{
        rpcServerManager: rpcServerManager,
    }
}

func (l *receiveDataListener) NewStream(streamId string, responseEncoder types.StreamEncoder) types.StreamDecoder {
    return newReceiveDataStreamDecoder(l.rpcServerManager)
}

func (l *receiveDataListener) OnGoAway() {

}

type receiveDataStreamDecoder struct {
    rpcServerManager servermanager.RPCServerManager
}

func newReceiveDataStreamDecoder(rpcServerManager servermanager.RPCServerManager) *receiveDataStreamDecoder {
    return &receiveDataStreamDecoder{
        rpcServerManager: rpcServerManager,
    }
}

func (d *receiveDataStreamDecoder) OnDecodeHeaders(headers map[string]string, endStream bool) {

}

func (d *receiveDataStreamDecoder) OnDecodeData(data types.IoBuffer, endStream bool) {
    if !endStream {
        return
    }
    receivedData := &model.ReceivedDataPb{}
    err := proto.Unmarshal(data.Bytes(), receivedData)
    if err != nil {
        log.DefaultLogger.Errorf("Unmarshal received data failed. error = %v", err)
        return
    }

    //cutoff @DEFAULT suffix
    receivedData.DataId = cutoffDataIdSuffix(receivedData.DataId)

    log.DefaultLogger.Infof("Received Confreg pushed data. data id = %s, segment = %s, version = %d, data = %v",
        receivedData.DataId, receivedData.Segment, receivedData.Version, receivedData.Data)

    if existedSubscriber(receivedData.DataId) {
        d.rpcServerManager.RegisterRPCServer(receivedData)
    }
}

func (d *receiveDataStreamDecoder) OnDecodeTrailers(trailers map[string]string) {

}