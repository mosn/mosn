package hessian

type ConnectionResponse struct {
	Ctx        ResponseContext
	Host       string
	Result     int32
	ErrorMsg   string
	ErrorStack string
}
