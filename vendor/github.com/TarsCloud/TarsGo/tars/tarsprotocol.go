package tars

import (
	"bytes"
	"context"
	"encoding/binary"
	"time"

	"github.com/TarsCloud/TarsGo/tars/util/current"

	"github.com/TarsCloud/TarsGo/tars/protocol/codec"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/basef"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
)

type dispatch interface {
	Dispatch(context.Context, interface{}, *requestf.RequestPacket, *requestf.ResponsePacket, bool) error
}

type TarsProtocol struct {
	dispatcher  dispatch
	serverImp   interface{}
	withContext bool
}

//NewTarsProtocol return a Tarsprotocol with dipatcher and implement interface.
//withContext explain using context or not.
func NewTarsProtocol(dispatcher dispatch, imp interface{}, withContext bool) *TarsProtocol {
	s := &TarsProtocol{dispatcher: dispatcher, serverImp: imp, withContext: withContext}
	return s
}

func (s *TarsProtocol) Invoke(ctx context.Context, req []byte) (rsp []byte) {
	defer checkPanic()
	reqPackage := requestf.RequestPacket{}
	rspPackage := requestf.ResponsePacket{}
	is := codec.NewReader(req)
	reqPackage.ReadFrom(is)
	TLOG.Debug("invoke:", reqPackage.IRequestId)
	if reqPackage.CPacketType == basef.TARSONEWAY {
		defer func() func() {
			beginTime := time.Now().UnixNano() / 1000000
			return func() {
				endTime := time.Now().UnixNano() / 1000000
				ReportStatFromServer(reqPackage.SFuncName, "one_way_client", rspPackage.IRet, endTime-beginTime)
			}
		}()()
	}
	var err error
	if s.withContext {
		ok := current.SetRequestStatus(ctx, reqPackage.Status)
		if !ok {
			TLOG.Error("Set reqeust status in context fail!")
		}
		ok = current.SetRequestContext(ctx, reqPackage.Context)
		if !ok {
			TLOG.Error("Set request context in context fail!")
		}
	}
	if allFilters.sf != nil {
		err = allFilters.sf(ctx, s.dispatcher.Dispatch, s.serverImp, &reqPackage, &rspPackage, s.withContext)
	} else {
		err = s.dispatcher.Dispatch(ctx, s.serverImp, &reqPackage, &rspPackage, s.withContext)
	}
	if err != nil {
		rspPackage.IRet = 1
		rspPackage.SResultDesc = err.Error()
	}
	return s.rsp2Byte(&rspPackage)
}

func (s *TarsProtocol) rsp2Byte(rsp *requestf.ResponsePacket) []byte {
	os := codec.NewBuffer()
	rsp.WriteTo(os)
	bs := os.ToBytes()
	sbuf := bytes.NewBuffer(nil)
	sbuf.Write(make([]byte, 4))
	sbuf.Write(bs)
	len := sbuf.Len()
	binary.BigEndian.PutUint32(sbuf.Bytes(), uint32(len))
	return sbuf.Bytes()
}

func (s *TarsProtocol) ParsePackage(buff []byte) (int, int) {
	return TarsRequest(buff)
}

func (s *TarsProtocol) InvokeTimeout(pkg []byte) []byte {
	rspPackage := requestf.ResponsePacket{}
	rspPackage.IRet = 1
	rspPackage.SResultDesc = "server invoke timeout"
	return s.rsp2Byte(&rspPackage)
}
