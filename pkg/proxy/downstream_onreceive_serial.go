// +build conn_serial

package proxy

import (
	"context"
	mosnctx "mosn.io/mosn/pkg/context"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/types"
)

// types.StreamReceiveListener
func (s *downStream) OnReceive(ctx context.Context, headers types.HeaderMap, data types.IoBuffer, trailers types.HeaderMap) {
	s.downstreamReqHeaders = headers
	s.context = mosnctx.WithValue(s.context, types.ContextKeyDownStreamHeaders, headers)
	s.downstreamReqDataBuf = data
	s.downstreamReqTrailers = trailers

	if log.Proxy.GetLogLevel() >= log.DEBUG {
		log.Proxy.Debugf(s.context, "[proxy] [downstream] OnReceive headers:%+v, data:%+v, trailers:%+v", headers, data, trailers)
	}

	id := s.ID

	task := func() {
		defer func() {
			if r := recover(); r != nil {
				log.Proxy.Alertf(s.context, types.ErrorKeyProxyPanic, "[proxy] [downstream] OnReceive panic: %v, downstream: %+v, oldId: %d, newId: %d\n%s",
					r, s, id, s.ID, string(debug.Stack()))

				if id == s.ID {
					s.cleanStream()
				}
			}
		}()

		phase := types.InitPhase
		for i := 0; i < 10; i++ {
			s.cleanNotify()

			phase = s.receive(ctx, id, phase)
			switch phase {
			case types.End:
				return
			case types.MatchRoute:
				log.Proxy.Debugf(s.context, "[proxy] [downstream] redo match route %+v", s)
			case types.Retry:
				log.Proxy.Debugf(s.context, "[proxy] [downstream] retry %+v", s)
			case types.UpFilter:
				log.Proxy.Debugf(s.context, "[proxy] [downstream] directResponse %+v", s)
			}
		}
	}

	s.proxy.readCallbacks.JobChan() <- task

	// goroutine for proxy
	s.proxy.readCallbacks.Once().Do(func(){
		go func() {
			for task := range s.proxy.readCallbacks.JobChan() {
				task()
			}
		}()
	})

}
