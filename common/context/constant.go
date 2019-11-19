package context

// ContextKey type
type ContextKey int

// Context key types(built-in)
const (
	ContextKeyStreamID ContextKey = iota
	ContextKeyConnectionID
	ContextKeyListenerPort
	ContextKeyListenerName
	ContextKeyListenerType
	ContextKeyListenerStatsNameSpace
	ContextKeyNetworkFilterChainFactories
	ContextKeyStreamFilterChainFactories
	ContextKeyBufferPoolCtx
	ContextKeyAccessLogs
	ContextOriRemoteAddr
	ContextKeyAcceptChan
	ContextKeyAcceptBuffer
	ContextKeyConnectionFd
	ContextSubProtocol
	ContextKeyTraceSpanKey
	ContextKeyActiveSpan
	ContextKeyTraceId
	ContextKeyEnd
)

// GlobalProxyName represents proxy name for metrics
const (
	GlobalProxyName = "global"
)
