package tars

import (
	"fmt"
	"github.com/TarsCloud/TarsGo/tars/protocol/res/statf"
	"net/http"
	"strings"
	"time"
)

var realIPHeader []string

func init() {
	realIPHeader = []string{ // the order is important
		"X-Real-Ip",
		"X-Forwarded-For-Pound",
		"X-Forwarded-For", //是一个列表，暂时全报吧
	}
}

type TarsHttpConf struct {
	Container string
	AppName   string
	IP        string
	Port      int32
	Version   string
	SetId     string
}

type TarsHttpMux struct {
	http.ServeMux
	cfg *TarsHttpConf
}

type httpStatInfo struct {
	reqAddr    string
	pattern    string
	statusCode int
	costTime   int64
}

func (mux *TarsHttpMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	h, pattern := mux.Handler(r)
	tw := &TarsResponseWriter{w, 0}
	startTime := time.Now().UnixNano() / 1000000
	h.ServeHTTP(tw, r)
	costTime := int64(time.Now().UnixNano()/1000000 - startTime)
	var reqAddr string
	for _, h := range realIPHeader {
		reqAddr = r.Header.Get(h)
		if reqAddr != "" {
			break
		}
	}
	if reqAddr == "" { // no proxy
		reqAddr = strings.SplitN(r.RemoteAddr, ":", 2)[0]
	}
	if pattern == "" {
		pattern = "/"
	}
	st := &httpStatInfo{
		reqAddr:    reqAddr,
		pattern:    pattern,
		statusCode: tw.statusCode,
		costTime:   costTime,
	}
	go mux.reportHttpStat(st)
}

func (mux *TarsHttpMux) reportHttpStat(st *httpStatInfo) {
	if mux.cfg == nil {
		return
	}
	cfg := mux.cfg
	var _statInfo = statf.StatMicMsgHead{}
	_statInfo.MasterName = "http_client"
	_statInfo.MasterIp = st.reqAddr

	_statInfo.TarsVersion = cfg.Version
	_statInfo.SlaveName = cfg.AppName
	_statInfo.SlaveIp = cfg.IP // from server
	_statInfo.SlavePort = cfg.Port
	//_statInfo.SSlaveContainer = cfg.Container
	_statInfo.InterfaceName = st.pattern
	if cfg.SetId != "" {
		setList := strings.Split(cfg.SetId, ".")
		_statInfo.SlaveSetName = setList[0]
		_statInfo.SlaveSetArea = setList[1]
		_statInfo.SlaveSetID = setList[2]
		//被调也要加上set信息
		_statInfo.SlaveName = fmt.Sprintf("http_client.%s%s%s", setList[0], setList[1], setList[2])
	}

	var _statBody = statf.StatMicMsgBody{}
	if st.statusCode >= 400 {
		_statBody.ExecCount = 1 // 异常
	} else {
		_statBody.Count = 1
		_statBody.TotalRspTime = st.costTime
		_statBody.MaxRspTime = int32(st.costTime)
		_statBody.MinRspTime = int32(st.costTime)
	}

	info := StatInfo{}
	info.Head = _statInfo
	info.Body = _statBody
	StatReport.pushBackMsg(info, true)
}

func (mux *TarsHttpMux) SetConfig(cfg *TarsHttpConf) {
	mux.cfg = cfg
}

type TarsResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *TarsResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
