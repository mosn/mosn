// Code generated by MockGen. DO NOT EDIT.
// Source: ../../vendor/mosn.io/api/stream_filter.go

// Package mock is a generated GoMock package.
package mock

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	api "mosn.io/api"
)

// MockStreamFilterBase is a mock of StreamFilterBase interface.
type MockStreamFilterBase struct {
	ctrl     *gomock.Controller
	recorder *MockStreamFilterBaseMockRecorder
}

// MockStreamFilterBaseMockRecorder is the mock recorder for MockStreamFilterBase.
type MockStreamFilterBaseMockRecorder struct {
	mock *MockStreamFilterBase
}

// NewMockStreamFilterBase creates a new mock instance.
func NewMockStreamFilterBase(ctrl *gomock.Controller) *MockStreamFilterBase {
	mock := &MockStreamFilterBase{ctrl: ctrl}
	mock.recorder = &MockStreamFilterBaseMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamFilterBase) EXPECT() *MockStreamFilterBaseMockRecorder {
	return m.recorder
}

// OnDestroy mocks base method.
func (m *MockStreamFilterBase) OnDestroy() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnDestroy")
}

// OnDestroy indicates an expected call of OnDestroy.
func (mr *MockStreamFilterBaseMockRecorder) OnDestroy() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnDestroy", reflect.TypeOf((*MockStreamFilterBase)(nil).OnDestroy))
}

// MockStreamSenderFilter is a mock of StreamSenderFilter interface.
type MockStreamSenderFilter struct {
	ctrl     *gomock.Controller
	recorder *MockStreamSenderFilterMockRecorder
}

// MockStreamSenderFilterMockRecorder is the mock recorder for MockStreamSenderFilter.
type MockStreamSenderFilterMockRecorder struct {
	mock *MockStreamSenderFilter
}

// NewMockStreamSenderFilter creates a new mock instance.
func NewMockStreamSenderFilter(ctrl *gomock.Controller) *MockStreamSenderFilter {
	mock := &MockStreamSenderFilter{ctrl: ctrl}
	mock.recorder = &MockStreamSenderFilterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamSenderFilter) EXPECT() *MockStreamSenderFilterMockRecorder {
	return m.recorder
}

// Append mocks base method.
func (m *MockStreamSenderFilter) Append(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Append", ctx, headers, buf, trailers)
	ret0, _ := ret[0].(api.StreamFilterStatus)
	return ret0
}

// Append indicates an expected call of Append.
func (mr *MockStreamSenderFilterMockRecorder) Append(ctx, headers, buf, trailers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Append", reflect.TypeOf((*MockStreamSenderFilter)(nil).Append), ctx, headers, buf, trailers)
}

// OnDestroy mocks base method.
func (m *MockStreamSenderFilter) OnDestroy() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnDestroy")
}

// OnDestroy indicates an expected call of OnDestroy.
func (mr *MockStreamSenderFilterMockRecorder) OnDestroy() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnDestroy", reflect.TypeOf((*MockStreamSenderFilter)(nil).OnDestroy))
}

// SetSenderFilterHandler mocks base method.
func (m *MockStreamSenderFilter) SetSenderFilterHandler(handler api.StreamSenderFilterHandler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetSenderFilterHandler", handler)
}

// SetSenderFilterHandler indicates an expected call of SetSenderFilterHandler.
func (mr *MockStreamSenderFilterMockRecorder) SetSenderFilterHandler(handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetSenderFilterHandler", reflect.TypeOf((*MockStreamSenderFilter)(nil).SetSenderFilterHandler), handler)
}

// MockStreamReceiverFilter is a mock of StreamReceiverFilter interface.
type MockStreamReceiverFilter struct {
	ctrl     *gomock.Controller
	recorder *MockStreamReceiverFilterMockRecorder
}

// MockStreamReceiverFilterMockRecorder is the mock recorder for MockStreamReceiverFilter.
type MockStreamReceiverFilterMockRecorder struct {
	mock *MockStreamReceiverFilter
}

// NewMockStreamReceiverFilter creates a new mock instance.
func NewMockStreamReceiverFilter(ctrl *gomock.Controller) *MockStreamReceiverFilter {
	mock := &MockStreamReceiverFilter{ctrl: ctrl}
	mock.recorder = &MockStreamReceiverFilterMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamReceiverFilter) EXPECT() *MockStreamReceiverFilterMockRecorder {
	return m.recorder
}

// OnDestroy mocks base method.
func (m *MockStreamReceiverFilter) OnDestroy() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnDestroy")
}

// OnDestroy indicates an expected call of OnDestroy.
func (mr *MockStreamReceiverFilterMockRecorder) OnDestroy() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnDestroy", reflect.TypeOf((*MockStreamReceiverFilter)(nil).OnDestroy))
}

// OnReceive mocks base method.
func (m *MockStreamReceiverFilter) OnReceive(ctx context.Context, headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) api.StreamFilterStatus {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "OnReceive", ctx, headers, buf, trailers)
	ret0, _ := ret[0].(api.StreamFilterStatus)
	return ret0
}

// OnReceive indicates an expected call of OnReceive.
func (mr *MockStreamReceiverFilterMockRecorder) OnReceive(ctx, headers, buf, trailers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnReceive", reflect.TypeOf((*MockStreamReceiverFilter)(nil).OnReceive), ctx, headers, buf, trailers)
}

// SetReceiveFilterHandler mocks base method.
func (m *MockStreamReceiverFilter) SetReceiveFilterHandler(handler api.StreamReceiverFilterHandler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetReceiveFilterHandler", handler)
}

// SetReceiveFilterHandler indicates an expected call of SetReceiveFilterHandler.
func (mr *MockStreamReceiverFilterMockRecorder) SetReceiveFilterHandler(handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetReceiveFilterHandler", reflect.TypeOf((*MockStreamReceiverFilter)(nil).SetReceiveFilterHandler), handler)
}

// MockStreamFilterHandler is a mock of StreamFilterHandler interface.
type MockStreamFilterHandler struct {
	ctrl     *gomock.Controller
	recorder *MockStreamFilterHandlerMockRecorder
}

// MockStreamFilterHandlerMockRecorder is the mock recorder for MockStreamFilterHandler.
type MockStreamFilterHandlerMockRecorder struct {
	mock *MockStreamFilterHandler
}

// NewMockStreamFilterHandler creates a new mock instance.
func NewMockStreamFilterHandler(ctrl *gomock.Controller) *MockStreamFilterHandler {
	mock := &MockStreamFilterHandler{ctrl: ctrl}
	mock.recorder = &MockStreamFilterHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamFilterHandler) EXPECT() *MockStreamFilterHandlerMockRecorder {
	return m.recorder
}

// Connection mocks base method.
func (m *MockStreamFilterHandler) Connection() api.Connection {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connection")
	ret0, _ := ret[0].(api.Connection)
	return ret0
}

// Connection indicates an expected call of Connection.
func (mr *MockStreamFilterHandlerMockRecorder) Connection() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connection", reflect.TypeOf((*MockStreamFilterHandler)(nil).Connection))
}

// RequestInfo mocks base method.
func (m *MockStreamFilterHandler) RequestInfo() api.RequestInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequestInfo")
	ret0, _ := ret[0].(api.RequestInfo)
	return ret0
}

// RequestInfo indicates an expected call of RequestInfo.
func (mr *MockStreamFilterHandlerMockRecorder) RequestInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequestInfo", reflect.TypeOf((*MockStreamFilterHandler)(nil).RequestInfo))
}

// Route mocks base method.
func (m *MockStreamFilterHandler) Route() api.Route {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Route")
	ret0, _ := ret[0].(api.Route)
	return ret0
}

// Route indicates an expected call of Route.
func (mr *MockStreamFilterHandlerMockRecorder) Route() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Route", reflect.TypeOf((*MockStreamFilterHandler)(nil).Route))
}

// MockStreamSenderFilterHandler is a mock of StreamSenderFilterHandler interface.
type MockStreamSenderFilterHandler struct {
	ctrl     *gomock.Controller
	recorder *MockStreamSenderFilterHandlerMockRecorder
}

// MockStreamSenderFilterHandlerMockRecorder is the mock recorder for MockStreamSenderFilterHandler.
type MockStreamSenderFilterHandlerMockRecorder struct {
	mock *MockStreamSenderFilterHandler
}

// NewMockStreamSenderFilterHandler creates a new mock instance.
func NewMockStreamSenderFilterHandler(ctrl *gomock.Controller) *MockStreamSenderFilterHandler {
	mock := &MockStreamSenderFilterHandler{ctrl: ctrl}
	mock.recorder = &MockStreamSenderFilterHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamSenderFilterHandler) EXPECT() *MockStreamSenderFilterHandlerMockRecorder {
	return m.recorder
}

// Connection mocks base method.
func (m *MockStreamSenderFilterHandler) Connection() api.Connection {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connection")
	ret0, _ := ret[0].(api.Connection)
	return ret0
}

// Connection indicates an expected call of Connection.
func (mr *MockStreamSenderFilterHandlerMockRecorder) Connection() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connection", reflect.TypeOf((*MockStreamSenderFilterHandler)(nil).Connection))
}

// GetResponseData mocks base method.
func (m *MockStreamSenderFilterHandler) GetResponseData() api.IoBuffer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResponseData")
	ret0, _ := ret[0].(api.IoBuffer)
	return ret0
}

// GetResponseData indicates an expected call of GetResponseData.
func (mr *MockStreamSenderFilterHandlerMockRecorder) GetResponseData() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResponseData", reflect.TypeOf((*MockStreamSenderFilterHandler)(nil).GetResponseData))
}

// GetResponseHeaders mocks base method.
func (m *MockStreamSenderFilterHandler) GetResponseHeaders() api.HeaderMap {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResponseHeaders")
	ret0, _ := ret[0].(api.HeaderMap)
	return ret0
}

// GetResponseHeaders indicates an expected call of GetResponseHeaders.
func (mr *MockStreamSenderFilterHandlerMockRecorder) GetResponseHeaders() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResponseHeaders", reflect.TypeOf((*MockStreamSenderFilterHandler)(nil).GetResponseHeaders))
}

// GetResponseTrailers mocks base method.
func (m *MockStreamSenderFilterHandler) GetResponseTrailers() api.HeaderMap {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetResponseTrailers")
	ret0, _ := ret[0].(api.HeaderMap)
	return ret0
}

// GetResponseTrailers indicates an expected call of GetResponseTrailers.
func (mr *MockStreamSenderFilterHandlerMockRecorder) GetResponseTrailers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResponseTrailers", reflect.TypeOf((*MockStreamSenderFilterHandler)(nil).GetResponseTrailers))
}

// RequestInfo mocks base method.
func (m *MockStreamSenderFilterHandler) RequestInfo() api.RequestInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequestInfo")
	ret0, _ := ret[0].(api.RequestInfo)
	return ret0
}

// RequestInfo indicates an expected call of RequestInfo.
func (mr *MockStreamSenderFilterHandlerMockRecorder) RequestInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequestInfo", reflect.TypeOf((*MockStreamSenderFilterHandler)(nil).RequestInfo))
}

// Route mocks base method.
func (m *MockStreamSenderFilterHandler) Route() api.Route {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Route")
	ret0, _ := ret[0].(api.Route)
	return ret0
}

// Route indicates an expected call of Route.
func (mr *MockStreamSenderFilterHandlerMockRecorder) Route() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Route", reflect.TypeOf((*MockStreamSenderFilterHandler)(nil).Route))
}

// SetResponseData mocks base method.
func (m *MockStreamSenderFilterHandler) SetResponseData(buf api.IoBuffer) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetResponseData", buf)
}

// SetResponseData indicates an expected call of SetResponseData.
func (mr *MockStreamSenderFilterHandlerMockRecorder) SetResponseData(buf interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetResponseData", reflect.TypeOf((*MockStreamSenderFilterHandler)(nil).SetResponseData), buf)
}

// SetResponseHeaders mocks base method.
func (m *MockStreamSenderFilterHandler) SetResponseHeaders(headers api.HeaderMap) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetResponseHeaders", headers)
}

// SetResponseHeaders indicates an expected call of SetResponseHeaders.
func (mr *MockStreamSenderFilterHandlerMockRecorder) SetResponseHeaders(headers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetResponseHeaders", reflect.TypeOf((*MockStreamSenderFilterHandler)(nil).SetResponseHeaders), headers)
}

// SetResponseTrailers mocks base method.
func (m *MockStreamSenderFilterHandler) SetResponseTrailers(trailers api.HeaderMap) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetResponseTrailers", trailers)
}

// SetResponseTrailers indicates an expected call of SetResponseTrailers.
func (mr *MockStreamSenderFilterHandlerMockRecorder) SetResponseTrailers(trailers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetResponseTrailers", reflect.TypeOf((*MockStreamSenderFilterHandler)(nil).SetResponseTrailers), trailers)
}

// MockStreamReceiverFilterHandler is a mock of StreamReceiverFilterHandler interface.
type MockStreamReceiverFilterHandler struct {
	ctrl     *gomock.Controller
	recorder *MockStreamReceiverFilterHandlerMockRecorder
}

// MockStreamReceiverFilterHandlerMockRecorder is the mock recorder for MockStreamReceiverFilterHandler.
type MockStreamReceiverFilterHandlerMockRecorder struct {
	mock *MockStreamReceiverFilterHandler
}

// NewMockStreamReceiverFilterHandler creates a new mock instance.
func NewMockStreamReceiverFilterHandler(ctrl *gomock.Controller) *MockStreamReceiverFilterHandler {
	mock := &MockStreamReceiverFilterHandler{ctrl: ctrl}
	mock.recorder = &MockStreamReceiverFilterHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamReceiverFilterHandler) EXPECT() *MockStreamReceiverFilterHandlerMockRecorder {
	return m.recorder
}

// AppendData mocks base method.
func (m *MockStreamReceiverFilterHandler) AppendData(buf api.IoBuffer, endStream bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AppendData", buf, endStream)
}

// AppendData indicates an expected call of AppendData.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) AppendData(buf, endStream interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppendData", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).AppendData), buf, endStream)
}

// AppendHeaders mocks base method.
func (m *MockStreamReceiverFilterHandler) AppendHeaders(headers api.HeaderMap, endStream bool) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AppendHeaders", headers, endStream)
}

// AppendHeaders indicates an expected call of AppendHeaders.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) AppendHeaders(headers, endStream interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppendHeaders", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).AppendHeaders), headers, endStream)
}

// AppendTrailers mocks base method.
func (m *MockStreamReceiverFilterHandler) AppendTrailers(trailers api.HeaderMap) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AppendTrailers", trailers)
}

// AppendTrailers indicates an expected call of AppendTrailers.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) AppendTrailers(trailers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AppendTrailers", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).AppendTrailers), trailers)
}

// Connection mocks base method.
func (m *MockStreamReceiverFilterHandler) Connection() api.Connection {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connection")
	ret0, _ := ret[0].(api.Connection)
	return ret0
}

// Connection indicates an expected call of Connection.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) Connection() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connection", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).Connection))
}

// GetFilterCurrentPhase mocks base method.
func (m *MockStreamReceiverFilterHandler) GetFilterCurrentPhase() api.ReceiverFilterPhase {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFilterCurrentPhase")
	ret0, _ := ret[0].(api.ReceiverFilterPhase)
	return ret0
}

// GetFilterCurrentPhase indicates an expected call of GetFilterCurrentPhase.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) GetFilterCurrentPhase() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFilterCurrentPhase", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).GetFilterCurrentPhase))
}

// GetRequestData mocks base method.
func (m *MockStreamReceiverFilterHandler) GetRequestData() api.IoBuffer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRequestData")
	ret0, _ := ret[0].(api.IoBuffer)
	return ret0
}

// GetRequestData indicates an expected call of GetRequestData.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) GetRequestData() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRequestData", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).GetRequestData))
}

// GetRequestHeaders mocks base method.
func (m *MockStreamReceiverFilterHandler) GetRequestHeaders() api.HeaderMap {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRequestHeaders")
	ret0, _ := ret[0].(api.HeaderMap)
	return ret0
}

// GetRequestHeaders indicates an expected call of GetRequestHeaders.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) GetRequestHeaders() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRequestHeaders", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).GetRequestHeaders))
}

// GetRequestTrailers mocks base method.
func (m *MockStreamReceiverFilterHandler) GetRequestTrailers() api.HeaderMap {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetRequestTrailers")
	ret0, _ := ret[0].(api.HeaderMap)
	return ret0
}

// GetRequestTrailers indicates an expected call of GetRequestTrailers.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) GetRequestTrailers() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetRequestTrailers", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).GetRequestTrailers))
}

// RequestInfo mocks base method.
func (m *MockStreamReceiverFilterHandler) RequestInfo() api.RequestInfo {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequestInfo")
	ret0, _ := ret[0].(api.RequestInfo)
	return ret0
}

// RequestInfo indicates an expected call of RequestInfo.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) RequestInfo() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequestInfo", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).RequestInfo))
}

// Route mocks base method.
func (m *MockStreamReceiverFilterHandler) Route() api.Route {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Route")
	ret0, _ := ret[0].(api.Route)
	return ret0
}

// Route indicates an expected call of Route.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) Route() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Route", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).Route))
}

// SendDirectResponse mocks base method.
func (m *MockStreamReceiverFilterHandler) SendDirectResponse(headers api.HeaderMap, buf api.IoBuffer, trailers api.HeaderMap) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendDirectResponse", headers, buf, trailers)
}

// SendDirectResponse indicates an expected call of SendDirectResponse.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) SendDirectResponse(headers, buf, trailers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendDirectResponse", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).SendDirectResponse), headers, buf, trailers)
}

// SendHijackReply mocks base method.
func (m *MockStreamReceiverFilterHandler) SendHijackReply(code int, headers api.HeaderMap) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendHijackReply", code, headers)
}

// SendHijackReply indicates an expected call of SendHijackReply.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) SendHijackReply(code, headers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHijackReply", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).SendHijackReply), code, headers)
}

// SendHijackReplyWithBody mocks base method.
func (m *MockStreamReceiverFilterHandler) SendHijackReplyWithBody(code int, headers api.HeaderMap, body string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SendHijackReplyWithBody", code, headers, body)
}

// SendHijackReplyWithBody indicates an expected call of SendHijackReplyWithBody.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) SendHijackReplyWithBody(code, headers, body interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendHijackReplyWithBody", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).SendHijackReplyWithBody), code, headers, body)
}

// SetRequestData mocks base method.
func (m *MockStreamReceiverFilterHandler) SetRequestData(buf api.IoBuffer) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetRequestData", buf)
}

// SetRequestData indicates an expected call of SetRequestData.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) SetRequestData(buf interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetRequestData", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).SetRequestData), buf)
}

// SetRequestHeaders mocks base method.
func (m *MockStreamReceiverFilterHandler) SetRequestHeaders(headers api.HeaderMap) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetRequestHeaders", headers)
}

// SetRequestHeaders indicates an expected call of SetRequestHeaders.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) SetRequestHeaders(headers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetRequestHeaders", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).SetRequestHeaders), headers)
}

// SetRequestTrailers mocks base method.
func (m *MockStreamReceiverFilterHandler) SetRequestTrailers(trailers api.HeaderMap) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SetRequestTrailers", trailers)
}

// SetRequestTrailers indicates an expected call of SetRequestTrailers.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) SetRequestTrailers(trailers interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetRequestTrailers", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).SetRequestTrailers), trailers)
}

// TerminateStream mocks base method.
func (m *MockStreamReceiverFilterHandler) TerminateStream(code int) bool {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TerminateStream", code)
	ret0, _ := ret[0].(bool)
	return ret0
}

// TerminateStream indicates an expected call of TerminateStream.
func (mr *MockStreamReceiverFilterHandlerMockRecorder) TerminateStream(code interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TerminateStream", reflect.TypeOf((*MockStreamReceiverFilterHandler)(nil).TerminateStream), code)
}

// MockStreamFilterChainFactory is a mock of StreamFilterChainFactory interface.
type MockStreamFilterChainFactory struct {
	ctrl     *gomock.Controller
	recorder *MockStreamFilterChainFactoryMockRecorder
}

// MockStreamFilterChainFactoryMockRecorder is the mock recorder for MockStreamFilterChainFactory.
type MockStreamFilterChainFactoryMockRecorder struct {
	mock *MockStreamFilterChainFactory
}

// NewMockStreamFilterChainFactory creates a new mock instance.
func NewMockStreamFilterChainFactory(ctrl *gomock.Controller) *MockStreamFilterChainFactory {
	mock := &MockStreamFilterChainFactory{ctrl: ctrl}
	mock.recorder = &MockStreamFilterChainFactoryMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamFilterChainFactory) EXPECT() *MockStreamFilterChainFactoryMockRecorder {
	return m.recorder
}

// CreateFilterChain mocks base method.
func (m *MockStreamFilterChainFactory) CreateFilterChain(context context.Context, callbacks api.StreamFilterChainFactoryCallbacks) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "CreateFilterChain", context, callbacks)
}

// CreateFilterChain indicates an expected call of CreateFilterChain.
func (mr *MockStreamFilterChainFactoryMockRecorder) CreateFilterChain(context, callbacks interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateFilterChain", reflect.TypeOf((*MockStreamFilterChainFactory)(nil).CreateFilterChain), context, callbacks)
}

// MockStreamFilterChainFactoryCallbacks is a mock of StreamFilterChainFactoryCallbacks interface.
type MockStreamFilterChainFactoryCallbacks struct {
	ctrl     *gomock.Controller
	recorder *MockStreamFilterChainFactoryCallbacksMockRecorder
}

// MockStreamFilterChainFactoryCallbacksMockRecorder is the mock recorder for MockStreamFilterChainFactoryCallbacks.
type MockStreamFilterChainFactoryCallbacksMockRecorder struct {
	mock *MockStreamFilterChainFactoryCallbacks
}

// NewMockStreamFilterChainFactoryCallbacks creates a new mock instance.
func NewMockStreamFilterChainFactoryCallbacks(ctrl *gomock.Controller) *MockStreamFilterChainFactoryCallbacks {
	mock := &MockStreamFilterChainFactoryCallbacks{ctrl: ctrl}
	mock.recorder = &MockStreamFilterChainFactoryCallbacksMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStreamFilterChainFactoryCallbacks) EXPECT() *MockStreamFilterChainFactoryCallbacksMockRecorder {
	return m.recorder
}

// AddStreamAccessLog mocks base method.
func (m *MockStreamFilterChainFactoryCallbacks) AddStreamAccessLog(accessLog api.AccessLog) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddStreamAccessLog", accessLog)
}

// AddStreamAccessLog indicates an expected call of AddStreamAccessLog.
func (mr *MockStreamFilterChainFactoryCallbacksMockRecorder) AddStreamAccessLog(accessLog interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddStreamAccessLog", reflect.TypeOf((*MockStreamFilterChainFactoryCallbacks)(nil).AddStreamAccessLog), accessLog)
}

// AddStreamReceiverFilter mocks base method.
func (m *MockStreamFilterChainFactoryCallbacks) AddStreamReceiverFilter(filter api.StreamReceiverFilter, p api.ReceiverFilterPhase) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddStreamReceiverFilter", filter, p)
}

// AddStreamReceiverFilter indicates an expected call of AddStreamReceiverFilter.
func (mr *MockStreamFilterChainFactoryCallbacksMockRecorder) AddStreamReceiverFilter(filter, p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddStreamReceiverFilter", reflect.TypeOf((*MockStreamFilterChainFactoryCallbacks)(nil).AddStreamReceiverFilter), filter, p)
}

// AddStreamSenderFilter mocks base method.
func (m *MockStreamFilterChainFactoryCallbacks) AddStreamSenderFilter(filter api.StreamSenderFilter, p api.SenderFilterPhase) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddStreamSenderFilter", filter, p)
}

// AddStreamSenderFilter indicates an expected call of AddStreamSenderFilter.
func (mr *MockStreamFilterChainFactoryCallbacksMockRecorder) AddStreamSenderFilter(filter, p interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddStreamSenderFilter", reflect.TypeOf((*MockStreamFilterChainFactoryCallbacks)(nil).AddStreamSenderFilter), filter, p)
}
