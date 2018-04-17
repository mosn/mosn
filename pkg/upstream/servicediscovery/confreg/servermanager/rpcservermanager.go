package servermanager

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
)

type segmentData struct {
    segment string
    version int64
    data    map[string][]string
}

func newSegmentData(receivedData *model.ReceivedDataPb) segmentData {
    svrData := receivedData.Data
    result := make(map[string][]string)
    for zone, svrs := range svrData {
        resolvedSrvs := make([]string, 0, len(svrs.Data))
        for _, svr := range svrs.Data {
            resolvedSrvs = append(resolvedSrvs, svr.Data)
        }
        result[zone] = resolvedSrvs
    }
    return segmentData{
        segment: receivedData.Segment,
        version: receivedData.Version,
        data:    result,
    }
}

type DefaultRPCServerManager struct {
    //dataId => segment => segmentData
    svrData map[string]map[string]segmentData

    listeners []RPCServerChangeListener
}

func NewRPCServerManager() RPCServerManager {
    RPCServerManager := &DefaultRPCServerManager{
        svrData:   make(map[string]map[string]segmentData),
        listeners: make([]RPCServerChangeListener, 0, 100),
    }

    return RPCServerManager
}

func (sm *DefaultRPCServerManager) RegisterRPCServer(receivedData *model.ReceivedDataPb) {
    dataId := receivedData.DataId
    storedDataIdContainer, ok := sm.svrData[dataId]
    if !ok {
        segment := newSegmentData(receivedData)
        var segments = make(map[string]segmentData)
        segments[receivedData.Segment] = segment
        sm.svrData[dataId] = segments

        go sm.triggerRPCServerChangeEvent(dataId)
    } else {
        storedSegment, ok := storedDataIdContainer[receivedData.Segment]
        if !ok || storedSegment.version < receivedData.Version {
            segment := newSegmentData(receivedData)
            storedDataIdContainer[receivedData.Segment] = segment

            go sm.triggerRPCServerChangeEvent(dataId)
        }
    }
}

func (sm *DefaultRPCServerManager) GetRPCServerList(dataId string) map[string][]string {
    result := make(map[string][]string)

    segments, ok := sm.svrData[dataId]
    if !ok {
        return result
    }

    for _, segmentData := range segments {
        for zone, srvs := range segmentData.data {
            zoneSrvs := srvs
            result[zone] = zoneSrvs
        }
    }

    return result
}

func (sm *DefaultRPCServerManager) GetRPCServerListByZone(dataId string, zone string) []string {
    segments, ok := sm.svrData[dataId]
    if !ok {
        return []string{}
    }

    for _, segmentData := range segments {
        for storedZone, srvs := range segmentData.data {
            if storedZone == zone {
                return srvs
            }
        }
    }

    return []string{}
}

func (sm *DefaultRPCServerManager) triggerRPCServerChangeEvent(dataId string) {
    for _, listener := range sm.listeners {
        listener.OnRPCServerChanged(dataId, sm.GetRPCServerList(dataId))
    }
}

func (sm *DefaultRPCServerManager) RegisterRPCServerChangeListener(listener RPCServerChangeListener) {
    sm.listeners = append(sm.listeners, listener)
}
