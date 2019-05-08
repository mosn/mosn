package tars

import "github.com/TarsCloud/TarsGo/tars/protocol/res/notifyf"

type NotifyHelper struct {
	comm *Communicator
	tn   *notifyf.Notify
	tm   notifyf.ReportInfo
}

func (n *NotifyHelper) SetNotifyInfo(comm *Communicator, notify string, app string, server string, container string) {
	n.comm = comm
	n.comm.SetProperty("netthread", 1)
	n.tn = new(notifyf.Notify)
	comm.StringToProxy(notify, n.tn)
	//TODO:params
	var set string
	if v, ok := comm.GetProperty("setdivision"); ok {
		set = v
	}
	n.tm = notifyf.ReportInfo{
		0,
		app,
		set,
		container,
		server,
		"",
		"",
		0,
	}
}

func (n *NotifyHelper) ReportNotifyInfo(info string) {
	n.tm.SMessage = info
	TLOG.Debug(n.tm)
	n.tn.ReportNotifyInfo(&n.tm)
}

func reportNotifyInfo(info string) {
	ha := new(NotifyHelper)
	comm := NewCommunicator()
	comm.SetProperty("netthread", 1)
	notify := GetServerConfig().notify
	app := GetServerConfig().App
	server := GetServerConfig().Server
	container := GetServerConfig().Container
	//container := ""
	ha.SetNotifyInfo(comm, notify, app, server, container)
	defer func() {
		if err := recover(); err != nil {
			TLOG.Debug(err)
		}
	}()
	ha.ReportNotifyInfo(info)
}
