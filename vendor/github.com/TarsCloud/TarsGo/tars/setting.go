package tars

import "time"

//MaxInvoke number of woker routine to handle client request
//zero means  no contorl ,just one goroutine for a client request.
//runtime.NumCpu() usually best performance in the benchmark.
var MaxInvoke = 0

const (
	//for now ,some option shuold update from remote config

	//TarsVersion is tars vesion
	TarsVersion string = "1.1.0"

	//server

	//AcceptTimeout accept timeout
	AcceptTimeout time.Duration = 500 * time.Millisecond
	//ReadTimeout zero for not set read deadline for Conn (better  performance)
	ReadTimeout time.Duration = 0 * time.Millisecond
	//WriteTimeout zero for not set write deadline for Conn (better performance)
	WriteTimeout time.Duration = 0 * time.Millisecond
	//HandleTimeout zero for not set deadline for invoke user interface (better performance)
	HandleTimeout time.Duration = 0 * time.Millisecond
	//IdleTimeout idle timeout
	IdleTimeout time.Duration = 600000 * time.Millisecond
	//ZombileTimeout zombile timeout
	ZombileTimeout time.Duration = time.Second * 10
	//QueueCap queue gap
	QueueCap int = 10000000

	//client

	//ClientQueueLen client queue length
	ClientQueueLen int = 10000
	//ClientIdleTimeout client idle timeout
	ClientIdleTimeout time.Duration = time.Second * 600
	//ClientReadTimeout client read timeout
	ClientReadTimeout time.Duration = time.Millisecond * 100
	//ClientWriteTimeout client write timeout
	ClientWriteTimeout time.Duration = time.Millisecond * 3000
	//ReqDefaultTimeout request default timeout
	ReqDefaultTimeout int32 = 3000
	//ObjQueueMax obj queue max number
	ObjQueueMax int32 = 10000

	//log
	remotelogBuff int = 500000
	//MaxlogOneTime is the max logs for reporting in one time.
	MaxlogOneTime       int = 2000
	defualtRotateN          = 10
	defaultRotateSizeMB     = 100

	//report

	//PropertyReportInterval property report interval
	PropertyReportInterval time.Duration = 10 * time.Second
	//StatReportInterval stat report interval
	StatReportInterval time.Duration = 10 * time.Second
	remoteLogInterval  time.Duration = 5 * time.Second

	//mainloop

	//MainLoopTicker main loop ticker
	MainLoopTicker time.Duration = 10 * time.Second

	//adapter

	//AdapterProxyTicker adapter proxy ticker
	AdapterProxyTicker time.Duration = 10 * time.Second
	//AdapterProxyResetCount adapter proxy reset count
	AdapterProxyResetCount int = 5

	//communicator default ,update from remote config
	refreshEndpointInterval int = 60000
	reportInterval          int = 10000
	//AsyncInvokeTimeout async invoke timeout
	AsyncInvokeTimeout int = 3000

	//tcp network config

	//TCPReadBuffer tcp read buffer length
	TCPReadBuffer = 128 * 1024 * 1024
	//TCPWriteBuffer tcp write buffer length
	TCPWriteBuffer = 128 * 1024 * 1024
	//TCPNoDelay set tcp no delay
	TCPNoDelay = false
)
