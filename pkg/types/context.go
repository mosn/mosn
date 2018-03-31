package types

type ContextKey string

const (
	ContextKeyConnectionId              ContextKey = "ConnectionId"
	ContextKeyListenerPort              ContextKey = "ListenerPort"
	ContextKeyListenerName              ContextKey = "ListenerName"
	ContextKeyListenerStatsNameSpace    ContextKey = "ListenerStatsNameSpace"
	ContextKeyNetworkFilterChainFactory ContextKey = "NetworkFilterChainFactory"
	ContextKeyStreamFilterChainFactory  ContextKey = "StreamFilterChainFactory"
)

const (
	GlobalStatsNamespace = ""
)
