package faulttolerance

type AlipayFaultToleranceFilter struct {
	Ip      string
	AppName string
}

func NewAlipayInvocationStatDimension(ip string, appName string) InvocationStatDimension {
	return AlipayFaultToleranceFilter{
		Ip:      ip,
		AppName: appName,
	}
}
