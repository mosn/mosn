package registry

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/types"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/servermanager"
    "fmt"
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
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
    fmt.Println("receive data header")
}

func (d *receiveDataStreamDecoder) OnDecodeData(data types.IoBuffer, endStream bool) {
    if endStream {
        //receivedData := &model.ReceivedDataPb{}
        //proto.Unmarshal(data.Bytes(), receivedData)
        d.rpcServerManager.RegisterRPCServer(assembleReceivedData())
    }
}

func (d *receiveDataStreamDecoder) OnDecodeTrailers(trailers map[string]string) {

}

var version = int64(1)

func assembleReceivedData() *model.ReceivedDataPb {
    dataBox := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"data1"}, {"data2"}, {"data3"}},
    }

    dataBox2 := &model.DataBoxesPb{
        Data: []*model.DataBoxPb{{"c1"}, {"c2"}, {"c3"}},
    }

    version++
    rd := &model.ReceivedDataPb{
        DataId:  "someDataId1",
        Segment: "s1",
        Data:    map[string]*model.DataBoxesPb{"zone1": dataBox, "zone2": dataBox2},
        Version: version,
    }

    return rd
}
