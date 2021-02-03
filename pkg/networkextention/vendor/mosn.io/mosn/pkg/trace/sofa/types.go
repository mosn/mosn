package sofa

// Const of tracing key
const (
	// old key
	TARGET_SERVICE_KEY = "sofa_head_target_service"
	// new key
	SERVICE_KEY             = "service"
	TARGET_METHOD           = "sofa_head_method_name"
	RPC_ID_KEY              = "rpc_trace_context.sofaRpcId"
	TRACER_ID_KEY           = "rpc_trace_context.sofaTraceId"
	CALLER_IP_KEY           = "rpc_trace_context.sofaCallerIp"
	CALLER_ZONE_KEY         = "rpc_trace_context.sofaCallerZone"
	APP_NAME                = "rpc_trace_context.sofaCallerApp"
	SOFA_TRACE_BAGGAGE_DATA = "rpc_trace_context.sofaPenAttrs"
	// http key
	HTTP_RPC_ID_KEY    = "SOFA-RpcId"
	HTTP_TRACER_ID_KEY = "SOFA-TraceId"
)
