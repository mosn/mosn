package types

type ContextType string

const (
	ConnectionId           ContextType = "ConnectionId"
	ListenerPort           ContextType = "ListenerPort"
	ListenerName           ContextType = "ListenerName"
	ListenerStatsNameSpace ContextType = "ListenerStatsNameSpace"
)

const (
	GlobalStatsNamespace = ""
)