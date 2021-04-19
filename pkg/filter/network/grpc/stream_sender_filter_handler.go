package grpc

import (
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

type grpcStreamSenderFilterHandler struct {
	sfc *grpcStreamFilterChain
}

func newStreamSenderFilterHandler(sfc *grpcStreamFilterChain) *grpcStreamSenderFilterHandler {
	f := &grpcStreamSenderFilterHandler{sfc}
	return f
}

func (g grpcStreamSenderFilterHandler) Route() api.Route {
	log.DefaultLogger.Warnf("Route() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) RequestInfo() api.RequestInfo {
	log.DefaultLogger.Warnf("RequestInfo() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) Connection() api.Connection {
	log.DefaultLogger.Warnf("Connection() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) GetResponseHeaders() api.HeaderMap {
	log.DefaultLogger.Warnf("GetResponseHeaders() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) SetResponseHeaders(headers api.HeaderMap) {
	log.DefaultLogger.Warnf("SetResponseHeaders() not implemented yet")
}

func (g grpcStreamSenderFilterHandler) GetResponseData() api.IoBuffer {
	log.DefaultLogger.Warnf("GetResponseData() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) SetResponseData(buf api.IoBuffer) {
	log.DefaultLogger.Warnf("SetResponseData() not implemented yet")
}

func (g grpcStreamSenderFilterHandler) GetResponseTrailers() api.HeaderMap {
	log.DefaultLogger.Warnf("GetResponseTrailers() not implemented yet")
	return nil
}

func (g grpcStreamSenderFilterHandler) SetResponseTrailers(trailers api.HeaderMap) {
	log.DefaultLogger.Warnf("SetResponseTrailers() not implemented yet")
}







