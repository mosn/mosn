package tars

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
	"github.com/TarsCloud/TarsGo/tars/util/tools"
)

type ServantProxy struct {
	sid     int32
	name    string
	comm    *Communicator
	obj     *ObjectProxy
	timeout int
}

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

func (s *ServantProxy) TarsSetTimeout(t int) {
	s.timeout = t
}

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
		ITimeout:     ReqDefaultTimeout,
		Context:      reqContext,
		Status:       status,
	}
	msg := &Message{Req: &req, Ser: s, Obj: s.obj}
	msg.Init()
	var err error
	if allFilters.cf != nil {
		err = allFilters.cf(ctx, msg, s.obj.Invoke, time.Duration(s.timeout)*time.Millisecond)
	}else{
		err = s.obj.Invoke(ctx, msg, time.Duration(s.timeout)*time.Millisecond)
	}
	if err != nil {
		TLOG.Error("Invoke error:", s.name, sFuncName, err.Error())
		//TODO report exec
		msg.End()
		ReportStat(msg, 0, 1, 0)
		return err
	}
	msg.End()
	*Resp = *msg.Resp
	//report
	ReportStat(msg, 1, 0, 0)
	return err
}

type ServantProxyFactory struct {
	objs map[string]*ServantProxy
	comm *Communicator
	fm   *sync.Mutex
}

func (o *ServantProxyFactory) Init(comm *Communicator) {
	o.fm = new(sync.Mutex)
	o.comm = comm
	o.objs = make(map[string]*ServantProxy)
}

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
