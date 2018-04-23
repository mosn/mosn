package servermanager

import (
    "gitlab.alipay-inc.com/afe/mosn/pkg/upstream/servicediscovery/confreg/model"
    "github.com/deckarep/golang-set"
    "gitlab.alipay-inc.com/afe/mosn/pkg/log"
    "encoding/json"
)

type segmentData struct {
    Segment string              `json:"segment"`
    Version int64               `json:"version"`
    Data    map[string][]string `json:"data"`
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
        Segment: receivedData.Segment,
        Version: receivedData.Version,
        Data:    result,
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
        if !ok || storedSegment.Version < receivedData.Version {
            segment := newSegmentData(receivedData)
            storedDataIdContainer[receivedData.Segment] = segment

            go sm.triggerRPCServerChangeEvent(dataId)
        } else {
            log.DefaultLogger.Infof("Received data which the version is lower than stored data, and will ignore the data."+
                "stored version = %d, received version = %d", storedSegment.Version, receivedData.Version)
        }
    }
}

func (sm *DefaultRPCServerManager) GetRPCServerList(dataId string) (servers map[string][]string, ok bool) {
    segments, ok := sm.svrData[dataId]
    if !ok {
        return nil, false
    }

    zoneMapServerSet := make(map[string]mapset.Set)
    for _, segmentData := range segments {
        for zone, srvs := range segmentData.Data {
            if zoneMapServerSet[zone] == nil {
                zoneMapServerSet[zone] = mapset.NewSet()
            }
            for _, srv := range srvs {
                zoneMapServerSet[zone].Add(srv)
            }
        }
    }

    result := make(map[string][]string)
    for zone, srvSet := range zoneMapServerSet {
        srvSetSlice := srvSet.ToSlice()
        result[zone] = make([]string, len(srvSetSlice))
        for i, srv := range srvSetSlice {
            result[zone][i] = srv.(string)
        }
    }

    return result, true
}

func (sm *DefaultRPCServerManager) GetRPCServerListByZone(dataId string, zone string) (servers []string, ok bool) {
    segments, ok := sm.svrData[dataId]
    if !ok {
        return nil, false
    }
    srvSet := mapset.NewSet()
    for _, segmentData := range segments {
        for segZone, srvs := range segmentData.Data {
            if segZone != zone {
                continue
            }
            for _, srv := range srvs {
                srvSet.Add(srv)
            }
        }
    }
    srvSetSlice := srvSet.ToSlice()
    if len(srvSetSlice) == 0 {
        return nil, false
    }

    srvs := make([]string, len(srvSetSlice))
    for i, srv := range srvSetSlice {
        srvs[i] = srv.(string)
    }

    return srvs, true
}

func (sm *DefaultRPCServerManager) GetRPCServiceSnapshot() []byte {
    snapshot, _ := json.Marshal(sm.svrData)
    return snapshot
}

func (sm *DefaultRPCServerManager) triggerRPCServerChangeEvent(dataId string) {
    data, _ := sm.GetRPCServerList(dataId)
    for _, listener := range sm.listeners {
        listener.OnRPCServerChanged(dataId, data)
    }
}

func (sm *DefaultRPCServerManager) RegisterRPCServerChangeListener(listener RPCServerChangeListener) {
    sm.listeners = append(sm.listeners, listener)
}
