package types



type AccessLog interface {
	Log(reqHeaders map[string]string, respHeaders map[string]string, requestInfo RequestInfo)
}

type AccessLogFilter interface {
	Decide(reqHeaders map[string]string, requestInfo RequestInfo) bool
}

type AccessLogFormatter interface {
	Format(reqHeaders map[string]string, respHeaders map[string]string, requestInfo RequestInfo) string
}