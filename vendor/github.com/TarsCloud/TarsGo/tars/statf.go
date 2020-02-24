package tars

import (
	"container/list"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/TarsCloud/TarsGo/tars/protocol/res/statf"
)

type StatInfo struct {
	Head statf.StatMicMsgHead
	Body statf.StatMicMsgBody
}

type StatFHelper struct {
	lStatInfo  *list.List
	mStatInfo  map[statf.StatMicMsgHead]statf.StatMicMsgBody
	mStatCount map[statf.StatMicMsgHead]int
	mlock      *sync.Mutex
	comm       *Communicator
	sf         *statf.StatF
	node       string

	lStatInfoFromServer *list.List
}

func (s *StatFHelper) Init(comm *Communicator, node string) {
	s.node = node
	s.lStatInfo = list.New()
	s.lStatInfoFromServer = list.New()
	s.mlock = new(sync.Mutex)
	s.mStatInfo = make(map[statf.StatMicMsgHead]statf.StatMicMsgBody)
	s.mStatCount = make(map[statf.StatMicMsgHead]int)
	s.comm = comm
	s.sf = new(statf.StatF)
	s.comm.StringToProxy(s.node, s.sf)
}

func (s *StatFHelper) addUpMsg(statList *list.List, fromServer bool) {
	s.mlock.Lock()
	var n *list.Element
	TLOG.Debug("report statList.size:", statList.Len())
	for e := statList.Front(); e != nil; e = n {
		statInfo := e.Value.(StatInfo)
		bodyList := statInfo.Body
		body, ok := s.mStatInfo[statInfo.Head]
		if ok {
			body.Count = (body.Count + statInfo.Body.Count)
			body.TimeoutCount = (body.TimeoutCount + statInfo.Body.TimeoutCount)
			body.ExecCount = (body.ExecCount + statInfo.Body.ExecCount)
			body.TotalRspTime = (body.TotalRspTime + statInfo.Body.TotalRspTime)
			body.MaxRspTime = (body.MaxRspTime + statInfo.Body.MaxRspTime)
			body.MinRspTime = (body.MinRspTime + statInfo.Body.MinRspTime)
			//body.WeightValue = (body.WeightValue + statInfo.Body.WeightValue)
			//body.WeightCount = (body.WeightCount + statInfo.Body.WeightCount)
			s.mStatInfo[statInfo.Head] = body
			s.mStatCount[statInfo.Head]++
		} else {
			headMap := statInfo.Head
			firstBody := statf.StatMicMsgBody{}
			firstBody.Count = bodyList.Count
			firstBody.TimeoutCount = bodyList.TimeoutCount
			firstBody.ExecCount = bodyList.ExecCount
			firstBody.TotalRspTime = bodyList.TotalRspTime
			firstBody.MaxRspTime = bodyList.MaxRspTime
			firstBody.MinRspTime = bodyList.MinRspTime
			//firstBody.WeightValue = bodyList.WeightValue
			//firstBody.WeightCount = bodyList.WeightCount
			s.mStatInfo[headMap] = firstBody
			s.mStatCount[headMap] = 1
		}

		n = e.Next()
		statList.Remove(e)
	}
	s.mlock.Unlock()
	ret, err := s.sf.ReportMicMsg(s.mStatInfo, !fromServer)
	if err != nil {
		TLOG.Debug("report err:", err.Error())
	}
	TLOG.Debug("report ret:", ret)
	s.mlock.Lock()
	for m := range s.mStatInfo {
		delete(s.mStatInfo, m)
	}
	s.mlock.Unlock()
}

func (s *StatFHelper) Run() {
	loop := time.NewTicker(StatReportInterval)
	for range loop.C {
		s.addUpMsg(s.lStatInfo, false)
		s.addUpMsg(s.lStatInfoFromServer, true)
		TLOG.Debug("stat report")
	}
}

func (s *StatFHelper) pushBackMsg(stStatInfo StatInfo, fromServer bool) {
	defer s.mlock.Unlock()
	s.mlock.Lock()
	if fromServer {
		s.lStatInfoFromServer.PushFront(stStatInfo)
	} else {
		s.lStatInfo.PushFront(stStatInfo)
	}
}

func (s *StatFHelper) ReportMicMsg(stStatInfo StatInfo, fromServer bool) {
	s.pushBackMsg(stStatInfo, fromServer)
}

var StatReport *StatFHelper
var statInited = make(chan struct{},1)

func initReport() {
	if GetClientConfig() == nil {
		statInited<-struct{}{}
		return
	}
	comm := NewCommunicator()
	comm.SetProperty("netthread", 1)
	StatReport = new(StatFHelper)
	StatReport.Init(comm, GetClientConfig().stat)
	statInited<-struct{}{}
	go StatReport.Run()
}

func ReportStatBase(head *statf.StatMicMsgHead, body *statf.StatMicMsgBody, FromServer bool) {
	cfg := GetServerConfig()
	statInfo := StatInfo{Head: *head, Body: *body}
	statInfo.Head.TarsVersion = cfg.Version
	//statInfo.Head.IStatVer = 2
	StatReport.ReportMicMsg(statInfo, FromServer)
}

func ReportStatFromClient(msg *Message, succ int32, timeout int32, exec int32) {
	cfg := GetServerConfig()
	if cfg == nil {
		return
	}
	var head statf.StatMicMsgHead
	var body statf.StatMicMsgBody
	head.MasterName = fmt.Sprintf("%s.%s", cfg.App, cfg.Server)
	head.MasterIp = cfg.LocalIP
	//head.SMasterContainer = cfg.Container
	if cfg.Enableset {
		setList := strings.Split(cfg.Setdivision, ".")
		head.MasterName = fmt.Sprintf("%s.%s.%s%s%s@%s", cfg.App, cfg.Server, setList[0], setList[1], setList[2], cfg.Version)
		//head.SMasterSetInfo = cfg.Setdivision
	}

	head.InterfaceName = msg.Req.SFuncName
	sNames := strings.Split(msg.Req.SServantName, ".")
	if len(sNames) < 2 {
		TLOG.Debug("report err:servant name (%s) format error", msg.Req.SServantName)
		return
	}
	head.SlaveName = fmt.Sprintf("%s.%s", sNames[0], sNames[1])
	if msg.Adp != nil {
		//head.SSlaveContainer = msg.Adp.GetPoint().ContainerName
		head.SlaveIp = msg.Adp.GetPoint().Host
		head.SlavePort = msg.Adp.GetPoint().Port
		if msg.Adp.GetPoint().SetId != "" {
			setList := strings.Split(msg.Adp.GetPoint().SetId, ".")
			head.SlaveSetName = setList[0]
			head.SlaveSetArea = setList[1]
			head.SlaveSetID = setList[2]
			head.SlaveName = fmt.Sprintf("%s.%s.%s%s%s", sNames[0], sNames[1], setList[0], setList[1], setList[2])
		}
	}
	if msg.Resp != nil {
		head.ReturnValue = msg.Resp.IRet
	} else {
		head.ReturnValue = -1
	}

	body.Count = succ
	body.TimeoutCount = timeout
	body.ExecCount = exec
	body.TotalRspTime = msg.Cost()
	body.MaxRspTime = int32(body.TotalRspTime)
	body.MinRspTime = int32(body.TotalRspTime)
	ReportStatBase(&head, &body, false)
}

func ReportStatFromServer(InterfaceName, MasterName string, ReturnValue int32, TotalRspTime int64) {
	cfg := GetServerConfig()
	var head statf.StatMicMsgHead
	var body statf.StatMicMsgBody
	head.SlaveName = fmt.Sprintf("%s.%s", cfg.App, cfg.Server)
	head.SlaveIp = cfg.LocalIP
	//head.SSlaveContainer = cfg.Container
	if cfg.Enableset {
		setList := strings.Split(cfg.Setdivision, ".")
		head.SlaveName = fmt.Sprintf("%s.%s.%s%s%s", cfg.App, cfg.Server, setList[0], setList[1], setList[2])
		head.SlaveSetName = setList[0]
		head.SlaveSetArea = setList[1]
		head.SlaveSetID = setList[2]
	}
	head.InterfaceName = InterfaceName
	head.MasterName = MasterName
	head.ReturnValue = ReturnValue

	if ReturnValue == 0 {
		body.Count = 1
	} else {
		body.ExecCount = 1
	}
	body.TotalRspTime = TotalRspTime
	body.MaxRspTime = int32(body.TotalRspTime)
	body.MinRspTime = int32(body.TotalRspTime)
	ReportStatBase(&head, &body, true)
}

func ReportStat(msg *Message, succ int32, timeout int32, exec int32) {
	ReportStatFromClient(msg, succ, timeout, exec)
}
