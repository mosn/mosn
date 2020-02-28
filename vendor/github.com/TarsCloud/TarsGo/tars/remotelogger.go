package tars

import (
	"github.com/TarsCloud/TarsGo/tars/protocol/res/logf"
)

type RemoteTimeWriter struct {
	logInfo       *logf.LogInfo
	logs          chan string
	logPtr        *logf.Log
	reportSuccPtr *PropertyReport
	reportFailPtr *PropertyReport
}

func NewRemoteTimeWriter() *RemoteTimeWriter {
	rw := new(RemoteTimeWriter)
	rw.logInfo = new(logf.LogInfo)
	logs := make(chan string, remotelogBuff)
	rw.logs = logs
	rw.logPtr = new(logf.Log)
	comm := NewCommunicator()
	node := GetServerConfig().log
	rw.EnableSufix(true)
	rw.EnablePrefix(true)
	rw.SetSeparator("|")
	comm.StringToProxy(node, rw.logPtr)
	go rw.Sync2remote()
	return rw
}

func (rw *RemoteTimeWriter) Sync2remote() {
	maxLen := MaxlogOneTime
	for {
		size := len(rw.logs)
		round := size / maxLen
		left := size % maxLen
		for i := 0; i < round && round != 0; i++ {
			v := make([]string, 0, maxLen)
			var log string
			for j := 0; j < maxLen; j++ {
				log = <-rw.logs
				v = append(v, log)
			}
			if len(v) != 0 {
				err := rw.sync2remote(v)
				if err != nil {
					TLOG.Error("sync to remote error")
					rw.reportFailPtr.Report(len(v))
				}
				rw.reportSuccPtr.Report(len(v))
			}

		}
		v := make([]string, 0, maxLen)
		var log string
		for k := 0; k < left && left != 0; k++ {
			log = <-rw.logs
			v = append(v, log)

		}
		if len(v) != 0 {
			err := rw.sync2remote(v)
			if err != nil {
				TLOG.Error("sync to remote error")
				rw.reportFailPtr.Report(len(v))
			}
			rw.reportSuccPtr.Report(len(v))
		}

	}
}

func (rw *RemoteTimeWriter) sync2remote(s []string) error {
	err := rw.logPtr.LoggerbyInfo(rw.logInfo, s)
	return err
}

func (rw *RemoteTimeWriter) InitServerInfo(app string, server string, filename string, setdivision string) {
	rw.logInfo.Appname = app
	rw.logInfo.Servername = server
	rw.logInfo.SFilename = filename
	rw.logInfo.Setdivision = setdivision
	serverInfo := app + "." + server + "." + filename
	failServerInfo := serverInfo + "_log_send_fail"
	failSum := NewSum()
	rw.reportFailPtr = CreatePropertyReport(failServerInfo, failSum)
	succServerInfo := serverInfo + "_log_send_succ"
	succSum := NewSum()
	rw.reportSuccPtr = CreatePropertyReport(succServerInfo, succSum)

}

func (rw *RemoteTimeWriter) EnableSufix(hasSufix bool) {
	rw.logInfo.BHasSufix = hasSufix
}
func (rw *RemoteTimeWriter) EnablePrefix(hasAppNamePrefix bool) {
	rw.logInfo.BHasAppNamePrefix = hasAppNamePrefix
}

func (rw *RemoteTimeWriter) SetFileNameConcatStr(s string) {
	rw.logInfo.SConcatStr = s

}
func (rw *RemoteTimeWriter) SetSeparator(s string) {
	rw.logInfo.SSepar = s
}
func (rw *RemoteTimeWriter) EnableSqarewrapper(hasSquareBracket bool) {
	rw.logInfo.BHasSquareBracket = hasSquareBracket
}
func (rw *RemoteTimeWriter) SetLogType(logType string) {
	rw.logInfo.SLogType = logType

}
func (rw *RemoteTimeWriter) InitFormat(s string) {
	rw.logInfo.SFormat = s
}

func (rw *RemoteTimeWriter) NeedPrefix() bool {
	return false
}

func (rw *RemoteTimeWriter) Write(b []byte) {
	s := string(b[:])
	select {
	case rw.logs <- s:
	default:
		TLOG.Error("remote log chan is full")

	}
}
