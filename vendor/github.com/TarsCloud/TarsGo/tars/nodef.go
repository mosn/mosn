package tars

import (
	"github.com/TarsCloud/TarsGo/tars/protocol/res/nodef"
	"os"
)

type NodeFHelper struct {
	comm *Communicator
	si   nodef.ServerInfo
	sf   *nodef.ServerF
}

func (n *NodeFHelper) SetNodeInfo(comm *Communicator, node string, app string, server string) {
	n.comm = comm
	n.sf = new(nodef.ServerF)
	comm.StringToProxy(node, n.sf)
	n.si = nodef.ServerInfo{
		app,
		server,
		int32(os.Getpid()),
		"",
		//"tars",
		//container,
	}
}

func (n *NodeFHelper) KeepAlive(adapter string) {
	n.si.Adapter = adapter
	_, err := n.sf.KeepAlive(&n.si)
	if err != nil {
		TLOG.Error("keepalive fail:", adapter)
	}
}

func (n *NodeFHelper) ReportVersion(version string) {
	_, err := n.sf.ReportVersion(n.si.Application, n.si.ServerName, version)
	if err != nil {
		TLOG.Error("report Version fail:")
	}
}
