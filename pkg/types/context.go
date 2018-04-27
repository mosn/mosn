package types

type ContextKey string

const (
	ContextKeyStreamId                   ContextKey = "StreamId"
	ContextKeyConnectionId               ContextKey = "ConnectionId"
	ContextKeyListenerPort               ContextKey = "ListenerPort"
	ContextKeyListenerName               ContextKey = "ListenerName"
	ContextKeyListenerStatsNameSpace     ContextKey = "ListenerStatsNameSpace"
	ContextKeyNetworkFilterChainFactory  ContextKey = "NetworkFilterChainFactory"
	ContextKeyStreamFilterChainFactories ContextKey = "StreamFilterChainFactory"
	ContextKeyConnectionCodecMapPool     ContextKey = "ContextKeyConnectionCodecMapPool"

	ContextKeyLogger     ContextKey = "Logger"
	ContextKeyAccessLogs ContextKey = "AccessLogs"
)

const (
	GlobalStatsNamespace = ""
)
