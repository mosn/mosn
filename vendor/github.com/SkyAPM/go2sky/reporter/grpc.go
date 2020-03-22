// Licensed to SkyAPM org under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. SkyAPM org licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package reporter

import (
	"context"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"

	"github.com/SkyAPM/go2sky"
	"github.com/SkyAPM/go2sky/internal/tool"
	"github.com/SkyAPM/go2sky/reporter/grpc/common"
	v2 "github.com/SkyAPM/go2sky/reporter/grpc/language-agent-v2"
	"github.com/SkyAPM/go2sky/reporter/grpc/register"
)

const (
	maxSendQueueSize    int32 = 30000
	defaultPingInterval       = 20 * time.Second
)

var (
	errServiceRegister  = errors.New("fail to register service")
	errInstanceRegister = errors.New("fail to instance service")
)

// NewGRPCReporter create a new reporter to send data to gRPC oap server. Only one backend address is allowed.
func NewGRPCReporter(serverAddr string, opts ...GRPCReporterOption) (go2sky.Reporter, error) {
	r := &gRPCReporter{
		logger:       log.New(os.Stderr, "go2sky-gRPC", log.LstdFlags),
		sendCh:       make(chan *common.UpstreamSegment, maxSendQueueSize),
		pingInterval: defaultPingInterval,
	}
	for _, o := range opts {
		o(r)
	}
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure()) //TODO add TLS
	if err != nil {
		return nil, err
	}
	r.conn = conn
	r.registerClient = register.NewRegisterClient(conn)
	r.pingClient = register.NewServiceInstancePingClient(conn)
	r.traceClient = v2.NewTraceSegmentReportServiceClient(r.conn)
	return r, nil
}

// GRPCReporterOption allows for functional options to adjust behaviour
// of a gRPC reporter to be created by NewGRPCReporter
type GRPCReporterOption func(r *gRPCReporter)

// WithLogger setup logger for gRPC reporter
func WithLogger(logger *log.Logger) GRPCReporterOption {
	return func(r *gRPCReporter) {
		r.logger = logger
	}
}

// WithPingInterval setup ping interval
func WithPingInterval(interval time.Duration) GRPCReporterOption {
	return func(r *gRPCReporter) {
		r.pingInterval = interval
	}
}

type gRPCReporter struct {
	serviceID      int32
	instanceID     int32
	instanceName   string
	logger         *log.Logger
	sendCh         chan *common.UpstreamSegment
	registerClient register.RegisterClient
	conn           *grpc.ClientConn
	traceClient    v2.TraceSegmentReportServiceClient
	pingClient     register.ServiceInstancePingClient
	pingInterval   time.Duration
}

func (r *gRPCReporter) Register(service string, instance string) (int32, int32, error) {
	r.retryRegister(func() error {
		return r.registerService(service)
	})
	r.retryRegister(func() error {
		return r.registerInstance(instance)
	})
	r.initSendPipeline()
	r.ping()
	return r.serviceID, r.instanceID, nil
}

type retryFunction func() error

func (r *gRPCReporter) retryRegister(f retryFunction) {
	for {
		err := f()
		if err == nil {
			break
		}
		r.logger.Printf("register error %v \n", err)
		time.Sleep(time.Second)
	}
}

func (r *gRPCReporter) registerService(name string) error {
	in := &register.Services{
		Services: []*register.Service{
			{
				ServiceName: name,
			},
		},
	}
	mapping, err := r.registerClient.DoServiceRegister(context.Background(), in)
	if err != nil {
		return err
	}
	if len(mapping.Services) < 1 {
		return errServiceRegister
	}
	r.serviceID = mapping.Services[0].Value
	r.logger.Printf("the id of service '%s' is %d", name, r.serviceID)
	return nil
}

func (r *gRPCReporter) registerInstance(name string) error {
	var props []*common.KeyStringValuePair

	if os.Getpid() > 0 {
		kv := &common.KeyStringValuePair{
			Key:   "process_no",
			Value: strconv.Itoa(os.Getpid()),
		}
		props = append(props, kv)
	}
	if hs, err := os.Hostname(); err == nil {
		if hs != "" {
			kv := &common.KeyStringValuePair{
				Key:   "host_name",
				Value: hs,
			}
			props = append(props, kv)
		}
	}
	language := &common.KeyStringValuePair{
		Key:   "language",
		Value: "go",
	}
	props = append(props, language)
	in := &register.ServiceInstances{
		Instances: []*register.ServiceInstance{
			{
				ServiceId:    r.serviceID,
				InstanceUUID: name,
				Time:         tool.Millisecond(time.Now()),
				Properties:   props,
			},
		},
	}
	mapping, err := r.registerClient.DoServiceInstanceRegister(context.Background(), in)
	if err != nil {
		return err
	}
	if len(mapping.ServiceInstances) < 1 {
		return errInstanceRegister
	}
	r.instanceID = mapping.ServiceInstances[0].Value
	r.instanceName = name
	r.logger.Printf("the id of instance '%s' id is %d", name, r.instanceID)
	return nil
}

func (r *gRPCReporter) Send(spans []go2sky.ReportedSpan) {
	spanSize := len(spans)
	if spanSize < 1 {
		return
	}
	rootSpan := spans[spanSize-1]
	segment := &common.UpstreamSegment{
		GlobalTraceIds: []*common.UniqueId{
			{
				IdParts: rootSpan.Context().TraceID,
			},
		},
	}
	segmentObject := &v2.SegmentObject{
		ServiceId:         r.serviceID,
		ServiceInstanceId: r.instanceID,
		TraceSegmentId: &common.UniqueId{
			IdParts: rootSpan.Context().SegmentID,
		},
		Spans: make([]*v2.SpanObjectV2, spanSize),
	}
	for i, s := range spans {
		spanCtx := s.Context()
		segmentObject.Spans[i] = &v2.SpanObjectV2{
			SpanId:        spanCtx.SpanID,
			ParentSpanId:  spanCtx.ParentSpanID,
			StartTime:     s.StartTime(),
			EndTime:       s.EndTime(),
			OperationName: s.OperationName(),
			Peer:          s.Peer(),
			SpanType:      s.SpanType(),
			SpanLayer:     s.SpanLayer(),
			IsError:       s.IsError(),
			Tags:          s.Tags(),
			Logs:          s.Logs(),
			ComponentId:   s.ComponentID(),
		}
		srr := make([]*v2.SegmentReference, 0)
		if i == (spanSize-1) && spanCtx.ParentSpanID > -1 {
			srr = append(srr, &v2.SegmentReference{
				ParentSpanId: spanCtx.ParentSpanID,
				ParentTraceSegmentId: &common.UniqueId{
					IdParts: spanCtx.ParentSegmentID,
				},
				ParentServiceInstanceId: r.instanceID,
				RefType:                 common.RefType_CrossThread,
			})
		}
		if len(s.Refs()) > 0 {
			for _, tc := range s.Refs() {
				srr = append(srr, &v2.SegmentReference{
					ParentSpanId: tc.ParentSpanID,
					ParentTraceSegmentId: &common.UniqueId{
						IdParts: tc.ParentSegmentID,
					},
					ParentServiceInstanceId: tc.ParentServiceInstanceID,
					EntryEndpoint:           tc.EntryEndpoint,
					EntryEndpointId:         tc.EntryEndpointID,
					EntryServiceInstanceId:  tc.EntryServiceInstanceID,
					NetworkAddress:          tc.NetworkAddress,
					NetworkAddressId:        tc.NetworkAddressID,
					ParentEndpoint:          tc.ParentEndpoint,
					ParentEndpointId:        tc.ParentEndpointID,
					RefType:                 common.RefType_CrossProcess,
				})
			}
		}
		segmentObject.Spans[i].Refs = srr
	}
	b, err := proto.Marshal(segmentObject)
	if err != nil {
		r.logger.Printf("marshal segment object err %v", err)
		return
	}
	segment.Segment = b
	select {
	case r.sendCh <- segment:
	default:
		r.logger.Printf("reach max send buffer")
	}
}

func (r *gRPCReporter) Close() {
	close(r.sendCh)
}

func (r *gRPCReporter) initSendPipeline() {
	if r.traceClient == nil {
		return
	}
	go func() {
	StreamLoop:
		for {
			stream, err := r.traceClient.Collect(context.Background())
			if err != nil {
				r.logger.Printf("open stream error %v", err)
				time.Sleep(5 * time.Second)
				continue StreamLoop
			}
			for s := range r.sendCh {
				err = stream.Send(s)
				if err != nil {
					r.logger.Printf("send segment error %v", err)
					r.closeStream(stream)
					continue StreamLoop
				}
			}
			r.closeStream(stream)
			if r.conn != nil {
				if err := r.conn.Close(); err != nil {
					r.logger.Print(err)
				}
			}
			break
		}
	}()
}

func (r *gRPCReporter) closeStream(stream v2.TraceSegmentReportService_CollectClient) {
	err := stream.CloseSend()
	if err != nil {
		r.logger.Printf("send closing error %v", err)
	}
}

func (r *gRPCReporter) ping() {
	if r.pingInterval < 0 || r.pingClient == nil {
		return
	}
	go func() {
		for {
			if r.conn.GetState() == connectivity.Shutdown {
				break
			}
			_, err := r.pingClient.DoPing(context.Background(), &register.ServiceInstancePingPkg{
				Time:                tool.Millisecond(time.Now()),
				ServiceInstanceId:   r.instanceID,
				ServiceInstanceUUID: r.instanceName,
			})
			if err != nil {
				r.logger.Printf("pinging error %v", err)
			}
			time.Sleep(r.pingInterval)
		}
	}()

}
