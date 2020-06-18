package tars

import "time"

//MaxInvoke number of woker routine to handle client request
//zero means  no contorl ,just one goroutine for a client request.
//runtime.NumCpu() usually best performance in the benchmark.
var MaxInvoke int32

const (
	//for now ,some option shuold update from remote config

	//TarsVersion is tars vesion
	TarsVersion string = "1.1.0"

	//server

	//AcceptTimeout accept timeout,defaultvalue is 500 milliseconds
	AcceptTimeout = 500
	//ReadTimeout zero millisecond for not set read deadline for Conn (better  performance)
	ReadTimeout = 0
	//WriteTimeout zero millisecond for not set write deadline for Conn (better performance)
	WriteTimeout = 0
	//HandleTimeout zero millisecond for not set deadline for invoke user interface (better performance)
	HandleTimeout = 0
	//IdleTimeout idle timeout,defaultvalue is 600000 milliseconds
	IdleTimeout = 600000
	//ZombileTimeout zombile timeout,defaultvalue is 10000 milliseconds
	ZombileTimeout = 10000
	//QueueCap queue gap
	QueueCap int = 10000000

	//client

	//ClientQueueLen client queue length
	ClientQueueLen int = 10000
	//ClientIdleTimeout client idle timeout,defaultvalue is 600000 milliseconds
	ClientIdleTimeout = 600000
	//ClientReadTimeout client read timeout,defaultvalue is 100 milliseconds
	ClientReadTimeout = 100
	//ClientWriteTimeout client write timeout,defaultvalue is 3000 milliseconds
	ClientWriteTimeout = 3000
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

	//PropertyReportInterval property report interval,defaultvalue is 10000 milliseconds
	PropertyReportInterval = 10000
	//StatReportInterval stat report interval,defaultvalue is 10000 milliseconds
	StatReportInterval               = 10000
	remoteLogInterval  time.Duration = 5 * time.Second

	//mainloop

	//MainLoopTicker main loop ticker,defaultvalue is 10000 milliseconds
	MainLoopTicker = 10000

	//adapter

	//AdapterProxyTicker adapter proxy ticker,defaultvalue is 10000 milliseconds
	AdapterProxyTicker = 10000
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

	//GracedownTimeout set timeout (second) for grace shutdown
	GracedownTimeout   = 60
	graceCheckInterval = time.Millisecond * 500
)
