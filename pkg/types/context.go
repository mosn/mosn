package types

type ContextKey string

const (
	ContextKeyConnectionId               ContextKey = "ConnectionId"
	ContextKeyListenerPort               ContextKey = "ListenerPort"
	ContextKeyListenerName               ContextKey = "ListenerName"
	ContextKeyListenerStatsNameSpace     ContextKey = "ListenerStatsNameSpace"
	ContextKeyNetworkFilterChainFactory  ContextKey = "NetworkFilterChainFactory"
	ContextKeyStreamFilterChainFactories ContextKey = "StreamFilterChainFactory"
	ContextKeyConnectionCodecBufferPool  ContextKey = "ContextKeyConnectionCodecBufferPool"
)

const (
	GlobalStatsNamespace = ""
)
