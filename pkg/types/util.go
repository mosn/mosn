package types

var reason2code = map[StreamResetReason]int{
	StreamConnectionSuccessed: SuccessCode,
	UpstreamGlobalTimeout:     TimeoutExceptionCode,
	UpstreamPerTryTimeout:     TimeoutExceptionCode,
	StreamOverflow:            UpstreamOverFlowCode,
	StreamRemoteReset:         NoHealthUpstreamCode,
	UpstreamReset:             NoHealthUpstreamCode,
	StreamLocalReset:          NoHealthUpstreamCode,
	StreamConnectionFailed:    NoHealthUpstreamCode,
}

// ConvertReasonToCode is convert the reason to a spec code.
func ConvertReasonToCode(reason StreamResetReason) int {
	if code, ok := reason2code[reason]; ok {
		return code
	}

	return InternalErrorCode
}
