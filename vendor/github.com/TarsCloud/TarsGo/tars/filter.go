package tars

import (
	"context"
	"time"

	"github.com/TarsCloud/TarsGo/tars/protocol/res/requestf"
)

type filters struct {
	cf ClientFilter
	sf ServerFilter
}

var allFilters = filters{nil, nil}

//Invoke is used for Invoke tars server service
type Invoke func(ctx context.Context, msg *Message, timeout time.Duration) (err error)

//RegisterClientFilter  registers the Client filter , and will be executed in every request.
func RegisterClientFilter(f ClientFilter) {
	allFilters.cf = f
}

//Dispatch server side Dispatch
type Dispatch func(context.Context, interface{}, *requestf.RequestPacket, *requestf.ResponsePacket, bool) error

//ServerFilter is used for add Filter for server dispatcher ,for implementing plugins like opentracing.
type ServerFilter func(ctx context.Context, d Dispatch, f interface{}, req *requestf.RequestPacket, resp *requestf.ResponsePacket, withContext bool) (err error)

//ClientFilter is used for filter request & response for client, for implementing plugins like opentracing
type ClientFilter func(ctx context.Context, msg *Message, invoke Invoke, timeout time.Duration) (err error)

//RegisterServerFilter register the server filter.
func RegisterServerFilter(f ServerFilter) {
	allFilters.sf = f
}
