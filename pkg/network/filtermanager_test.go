package network

import (
	"testing"

	"mosn.io/api"
	"mosn.io/mosn/pkg/types"
	"mosn.io/pkg/buffer"
)

type testFilter struct{
	read bool
	write bool
	init bool
	readCallbacks api.ReadFilterCallbacks
}

func (tf *testFilter) OnData(buffer types.IoBuffer) api.FilterStatus {
	tf.read = true
 	return api.Continue
}

func (tf *testFilter) OnNewConnection() api.FilterStatus {
	tf.init = true
	tf.readCallbacks.Connection().GetReadBuffer().WriteByte(1)
	return api.Continue
}

func (tf *testFilter) InitializeReadFilterCallbacks(cb api.ReadFilterCallbacks) {
	tf.readCallbacks = cb
	var host api.HostInfo
	tf.readCallbacks.SetUpstreamHost(host)
	if host == tf.readCallbacks.UpstreamHost() {
		tf.readCallbacks.Connection().GetReadBuffer().WriteByte(1)
	}
}


func (tf *testFilter) OnWrite(buf []buffer.IoBuffer) api.FilterStatus {
	tf.write = true
	return api.Continue
}


func Test_filtermgr(t *testing.T) {
	conn := &connection{}
	conn.readBuffer = buffer.GetIoBuffer(10)
	conn.readBuffer.WriteByte(1)

	fm := newFilterManager(conn)
	tf := &testFilter{}
	fm.AddReadFilter(tf)
	fm.AddWriteFilter(tf)

	lrf := fm.ListReadFilter()
	if len(lrf) != 1 || lrf[0].(*testFilter) != tf{
		t.Errorf("list readfilter error")
		return
	}

	lwf := fm.ListWriteFilters()
	if len(lwf) != 1 || lrf[0].(*testFilter) != tf {
		t.Errorf("list writefilter error")
		return
	}
	fm.InitializeReadFilters()
	fm.OnRead()
	fm.OnWrite(nil)

    if !tf.read || !tf.write || !tf.init  {
    	t.Errorf("filtermgr error")
	}

	if conn.readBuffer.Len() != 3 {
		t.Errorf("filtermgr error")
	}
}