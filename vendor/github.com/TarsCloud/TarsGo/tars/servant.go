package tars

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TarsCloud/TarsGo/tars/protocol/res/basef"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	"github.com/TarsCloud/TarsGo/tars/util/tools"
)

//ServantProxy is the struct for proxy servants.
type ServantProxy struct {
	sid     int32
	name    string
	comm    *Communicator
	obj     *ObjectProxy
	timeout int
}

//Init init the ServantProxy struct.
func (s *ServantProxy) Init(comm *Communicator, objName string) {
	pos := strings.Index(objName, "@")
	if pos > 0 {
		s.name = objName[0:pos]
	} else {
		s.name = objName
	}
	s.comm = comm
	of := new(ObjectProxyFactory)
	of.Init(comm)
	s.timeout = s.comm.Client.AsyncInvokeTimeout
	s.obj = of.GetObjectProxy(objName)
}

//TarsSetTimeout sets the timeout for client calling the server , which is in ms.
func (s *ServantProxy) TarsSetTimeout(t int) {
	s.timeout = t
}

//Tars_invoke is use for client inoking server.
func (s *ServantProxy) Tars_invoke(ctx context.Context, ctype byte,
	sFuncName string,
	buf []byte,
	status map[string]string,
	reqContext map[string]string,
	Resp *requestf.ResponsePacket) error {
	defer checkPanic()
	//TODO 重置sid，防止溢出
	atomic.CompareAndSwapInt32(&s.sid, 1<<31-1, 1)
	req := requestf.RequestPacket{
		IVersion:     1,
		CPacketType:  0,
		IRequestId:   atomic.AddInt32(&s.sid, 1),
		SServantName: s.name,
		SFuncName:    sFuncName,
		SBuffer:      tools.ByteToInt8(buf),
		ITimeout:     s.comm.Client.ReqDefaultTimeout,
		Context:      reqContext,
		Status:       status,
	}
	msg := &Message{Req: &req, Ser: s, Obj: s.obj}
	msg.Init()
	var err error
	if allFilters.cf != nil {
		err = allFilters.cf(ctx, msg, s.obj.Invoke, time.Duration(s.timeout)*time.Millisecond)
	} else {
		err = s.obj.Invoke(ctx, msg, time.Duration(s.timeout)*time.Millisecond)
	}
	if err != nil {
		TLOG.Errorf("Invoke Obj:%s,fun:%s,error:%s", s.name, sFuncName, err.Error())
		if msg.Resp == nil {
			ReportStat(msg, 0, 0, 1)
		} else if msg.Status == basef.TARSINVOKETIMEOUT {
			ReportStat(msg, 0, 1, 0)
		} else {
			ReportStat(msg, 0, 0, 1)
		}
		return err
	}
	msg.End()
	*Resp = *msg.Resp
	//report
	ReportStat(msg, 1, 0, 0)
	return err
}

//ServantProxyFactory is ServantProxy' factory struct.
type ServantProxyFactory struct {
	objs map[string]*ServantProxy
	comm *Communicator
	fm   *sync.Mutex
}

//Init init the  ServantProxyFactory.
func (o *ServantProxyFactory) Init(comm *Communicator) {
	o.fm = new(sync.Mutex)
	o.comm = comm
	o.objs = make(map[string]*ServantProxy)
}

//GetServantProxy gets the ServanrProxy for the object.
func (o *ServantProxyFactory) GetServantProxy(objName string) *ServantProxy {
	o.fm.Lock()
	if obj, ok := o.objs[objName]; ok {
		o.fm.Unlock()
		return obj
	}
	o.fm.Unlock()
	obj := new(ServantProxy)
	obj.Init(o.comm, objName)
	o.fm.Lock()
	o.objs[objName] = obj
	o.fm.Unlock()
	return obj
}
