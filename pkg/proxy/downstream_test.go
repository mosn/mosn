package proxy

import (
	"context"
	"github.com/alipay/sofa-mosn/pkg/network"
	"github.com/alipay/sofa-mosn/pkg/trace"
	"github.com/alipay/sofa-mosn/pkg/types"
	"strconv"
	"testing"
	"time"
)

func TestDownstream_FinishTracing_NotEnable(t *testing.T) {
	ds := downStream{}
	ds.finishTracing(strconv.Itoa(types.SuccessCode))
}

func TestDownstream_FinishTracing_Enable(t *testing.T) {
	trace.EnableTracing()
	ds := downStream{context: context.Background()}
	ds.finishTracing(strconv.Itoa(types.SuccessCode))
}

func TestDownstream_FinishTracing_Enable_SpanIsNotNil(t *testing.T) {
	trace.EnableTracing()
	trace.CreateInstance()
	span := trace.SofaTracerInstance.Start(time.Now())
	ctx := context.WithValue(context.Background(), trace.ActiveSpanKey, span)
	requestInfo := &network.RequestInfo{
	}
	ds := downStream{context: ctx, requestInfo: requestInfo}
	ds.finishTracing(strconv.Itoa(types.SuccessCode))
}
