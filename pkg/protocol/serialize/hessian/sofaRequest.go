package hessian

type SofaRequest struct {
	TargetAppName           string
	MethodName              string
	TargetServiceUniqueName string
	RequestProps            map[string]interface{}
	MethodArgSigs           []string
}

/*
func (sofaRequest *SofaRequest) SettargetAppName(targetAppName string) {

	sofaRequest.targetAppName = targetAppName

}

func (sofaRequest *SofaRequest) SetmethodName(methodName string) {

	sofaRequest.methodName = methodName

}

func (sofaRequest *SofaRequest) SettargetServiceUniqueName(targetServiceUniqueName string) {

	sofaRequest.targetServiceUniqueName = targetServiceUniqueName

}


func (sofaRequest *SofaRequest) SettargetrequestProps(requestProps map[string]interface{}) {

	sofaRequest.requestProps = requestProps

}

func (sofaRequest *SofaRequest) SetmethodArgSigs(methodArgSigs []string) {

	sofaRequest.methodArgSigs = methodArgSigs

}*/
