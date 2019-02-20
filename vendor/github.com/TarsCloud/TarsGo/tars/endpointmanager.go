package tars

import (
	"strings"
	"sync"
	"time"

	"github.com/TarsCloud/TarsGo/tars/protocol/res/endpointf"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/queryf"
	"github.com/TarsCloud/TarsGo/tars/util/endpoint"
	"github.com/TarsCloud/TarsGo/tars/util/set"
)

// EndpointManager is a struct which contains endpoint information.
type EndpointManager struct {
	objName         string
	directproxy     bool
	adapters        map[endpoint.Endpoint]*AdapterProxy
	index           []interface{} //cache the set
	pointsSet       *set.Set
	comm            *Communicator
	mlock           *sync.Mutex
	refreshInterval int
	pos             int32
	depth           int32
}

func (e *EndpointManager) setObjName(objName string) {
	if objName == "" {
		return
	}
	pos := strings.Index(objName, "@")
	if pos > 0 {
		//[direct]
		e.objName = objName[0:pos]
		endpoints := objName[pos+1:]
		e.directproxy = true
		for _, end := range strings.Split(endpoints, ":") {
			e.pointsSet.Add(endpoint.Parse(end))
		}
		e.index = e.pointsSet.Slice()

	} else {
		//[proxy] TODO singleton
		TLOG.Debug("proxy mode:", objName)
		e.objName = objName
		//comm := NewCommunicator()
		//comm.SetProperty("netthread", 1)
		obj, _ := e.comm.GetProperty("locator")
		q := new(queryf.QueryF)
		e.comm.StringToProxy(obj, q)
		e.findAndSetObj(q)
		go func() {
			loop := time.NewTicker(time.Duration(e.refreshInterval) * time.Millisecond)
			for range loop.C {
				//TODO exit
				e.findAndSetObj(q)
			}
		}()
	}
}

// Init endpoint struct.
func (e *EndpointManager) Init(objName string, comm *Communicator) error {
	e.comm = comm
	e.mlock = new(sync.Mutex)
	e.adapters = make(map[endpoint.Endpoint]*AdapterProxy)
	e.pointsSet = set.NewSet()
	e.directproxy = false
	e.refreshInterval = comm.Client.refreshEndpointInterval
	e.pos = 0
	e.depth = 0
	//ObjName要放到最后初始化
	e.setObjName(objName)
	return nil
}

// GetNextValidProxy returns polling adapter information.
func (e *EndpointManager) GetNextValidProxy() *AdapterProxy {
	e.mlock.Lock()
	ep := e.GetNextEndpoint()
	if ep == nil {
		e.mlock.Unlock()
		return nil
	}
	if adp, ok := e.adapters[*ep]; ok {
		// returns nil if recursively all nodes have not found an available node.
		if adp.status {
			e.mlock.Unlock()
			return adp
		} else if e.depth > e.pointsSet.Len() {
			e.mlock.Unlock()
			return nil
		} else {
			e.depth++
			e.mlock.Unlock()
			return e.GetNextValidProxy()
		}
	}
	err := e.createProxy(*ep)
	if err != nil {
		TLOG.Error("create adapter fail:", *ep, err)
		e.mlock.Unlock()
		return nil
	}
	a := e.adapters[*ep]
	e.mlock.Unlock()
	return a
}

// GetNextEndpoint returns the endpoint basic information.
func (e *EndpointManager) GetNextEndpoint() *endpoint.Endpoint {
	length := len(e.index)
	if length <= 0 {
		return nil
	}
	var ep endpoint.Endpoint
	e.pos = (e.pos + 1) % int32(length)
	ep = e.index[e.pos].(endpoint.Endpoint)
	return &ep
}

// GetAllEndpoint returns all endpoint information as a array(support not tars service).
func (e *EndpointManager) GetAllEndpoint() []*endpoint.Endpoint {
	es := make([]*endpoint.Endpoint, len(e.index))
	e.mlock.Lock()
	defer e.mlock.Unlock()
	for i, v := range e.index {
		e := v.(endpoint.Endpoint)
		es[i] = &e
	}
	return es
}

func (e *EndpointManager) createProxy(ep endpoint.Endpoint) error {
	TLOG.Debug("create adapter:", ep)
	adp := new(AdapterProxy)
	//TODO
	end := endpoint.Endpoint2tars(ep)
	err := adp.New(&end, e.comm)
	if err != nil {
		return err
	}
	e.adapters[ep] = adp
	return nil
}

// GetHashProxy returns hash adapter information.
func (e *EndpointManager) GetHashProxy(hashcode int64) *AdapterProxy {
	// very unsafe.
	ep := e.GetHashEndpoint(hashcode)
	if ep == nil {
		return nil
	}
	if adp, ok := e.adapters[*ep]; ok {
		return adp
	}
	err := e.createProxy(*ep)
	if err != nil {
		TLOG.Error("create adapter fail:", ep, err)
		return nil
	}
	return e.adapters[*ep]
}

// GetHashEndpoint returns hash endpoint information.
func (e *EndpointManager) GetHashEndpoint(hashcode int64) *endpoint.Endpoint {
	length := len(e.index)
	if length <= 0 {
		return nil
	}
	pos := hashcode % int64(length)
	ep := e.index[pos].(endpoint.Endpoint)
	return &ep
}

// SelectAdapterProxy returns selected adapter.
func (e *EndpointManager) SelectAdapterProxy(msg *Message) *AdapterProxy {
	if msg.isHash {
		return e.GetHashProxy(msg.hashCode)
	}
	return e.GetNextValidProxy()
}

func (e *EndpointManager) findAndSetObj(q *queryf.QueryF) {
	activeEp := new([]endpointf.EndpointF)
	inactiveEp := new([]endpointf.EndpointF)
	var setable, ok bool
	var setID string
	var ret int32
	var err error
	if setable, ok = e.comm.GetPropertyBool("enableset"); ok {
		setID, _ = e.comm.GetProperty("setdivision")
	}
	if setable {
		ret, err = q.FindObjectByIdInSameSet(e.objName, setID, activeEp, inactiveEp)
	} else {
		ret, err = q.FindObjectByIdInSameGroup(e.objName, activeEp, inactiveEp)
	}
	if err != nil {
		TLOG.Error("find obj end fail:", err.Error())
		return
	}
	TLOG.Debug("find obj endpoint:", e.objName, ret, *activeEp, *inactiveEp)

	e.mlock.Lock()
	if (len(*inactiveEp)) > 0 {
		for _, ep := range *inactiveEp {
			end := endpoint.Tars2endpoint(ep)
			e.pointsSet.Remove(end)
			if a, ok := e.adapters[end]; ok {
				delete(e.adapters, end)
				a.Close()
			}
		}
	}
	if (len(*activeEp)) > 0 {
		e.pointsSet.Clear() // clean it first,then add back .this action must lead to add lock,but if don't clean may lead to leakage.it's better to use remove.
		for _, ep := range *activeEp {
			end := endpoint.Tars2endpoint(ep)
			e.pointsSet.Add(end)
		}
		e.index = e.pointsSet.Slice()
	}
	for end := range e.adapters {
		// clean up dirty data
		if !e.pointsSet.Has(end) {
			if a, ok := e.adapters[end]; ok {
				delete(e.adapters, end)
				a.Close()
			}
		}
	}
	e.mlock.Unlock()
}
