package proxywasm_0_1_0

import (
	"net/http"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
	"mosn.io/mosn/pkg/mock"
	_ "mosn.io/mosn/pkg/stream/http"
	"mosn.io/pkg/buffer"
	"mosn.io/pkg/header"
)

type mockImportsHandler struct {
	*DefaultImportsHandler

	getRootContextIDPtr       func() int32
	getVmConfigPtr            func() buffer.IoBuffer
	getPluginConfigPtr        func() buffer.IoBuffer
	logPtr                    func(level log.Level, msg string)
	getHttpRequestHeaderPtr   func() api.HeaderMap
	getHttpRequestBodyPtr     func() buffer.IoBuffer
	getHttpRequestTrailerPtr  func() api.HeaderMap
	getHttpResponseHeaderPtr  func() api.HeaderMap
	getHttpResponseBodyPtr    func() buffer.IoBuffer
	getHttpResponseTrailerPtr func() api.HeaderMap
}

func newMockImportsHandler() *mockImportsHandler {
	return &mockImportsHandler{
		DefaultImportsHandler: &DefaultImportsHandler{},
	}
}

func (m *mockImportsHandler) GetRootContextID() int32 {
	if m.getRootContextIDPtr != nil {
		return m.getRootContextIDPtr()
	}
	return m.DefaultImportsHandler.GetRootContextID()
}

func (m *mockImportsHandler) GetVmConfig() buffer.IoBuffer {
	if m.getVmConfigPtr != nil {
		return m.getVmConfigPtr()
	}
	return m.DefaultImportsHandler.GetVmConfig()
}

func (m *mockImportsHandler) GetPluginConfig() buffer.IoBuffer {
	if m.getPluginConfigPtr != nil {
		return m.getPluginConfigPtr()
	}
	return m.DefaultImportsHandler.GetPluginConfig()
}

func (m *mockImportsHandler) Log(level log.Level, msg string) {
	if m.logPtr != nil {
		m.logPtr(level, msg)
	}
	m.DefaultImportsHandler.Log(level, msg)
}

func (m *mockImportsHandler) GetHttpRequestHeader() api.HeaderMap {
	if m.getHttpRequestHeaderPtr != nil {
		return m.getHttpRequestHeaderPtr()
	}
	return m.DefaultImportsHandler.GetHttpRequestHeader()
}

func (m *mockImportsHandler) GetHttpRequestBody() buffer.IoBuffer {
	if m.getHttpRequestBodyPtr != nil {
		return m.getHttpRequestBodyPtr()
	}
	return m.DefaultImportsHandler.GetHttpRequestBody()
}

func (m *mockImportsHandler) GetHttpRequestTrailer() api.HeaderMap {
	if m.getHttpRequestTrailerPtr != nil {
		return m.getHttpRequestTrailerPtr()
	}
	return m.DefaultImportsHandler.GetHttpRequestTrailer()
}

func (m *mockImportsHandler) GetHttpResponseHeader() api.HeaderMap {
	if m.getHttpResponseHeaderPtr != nil {
		return m.getHttpResponseHeaderPtr()
	}
	return m.DefaultImportsHandler.GetHttpResponseHeader()
}

func (m *mockImportsHandler) GetHttpResponseBody() buffer.IoBuffer {
	if m.getHttpResponseBodyPtr != nil {
		return m.getHttpResponseBodyPtr()
	}
	return m.DefaultImportsHandler.GetHttpResponseBody()
}

func (m *mockImportsHandler) GetHttpResponseTrailer() api.HeaderMap {
	if m.getHttpResponseTrailerPtr != nil {
		return m.getHttpResponseTrailerPtr()
	}
	return m.DefaultImportsHandler.GetHttpResponseTrailer()
}

func TestGetBuffer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	imports := newMockImportsHandler()
	imports.getVmConfigPtr = func() buffer.IoBuffer { return buffer.NewIoBufferString("vm config") }
	imports.getPluginConfigPtr = func() buffer.IoBuffer { return buffer.NewIoBufferString("plugin config") }

	imports.getHttpRequestBodyPtr = func() buffer.IoBuffer { return buffer.NewIoBufferString("req body") }
	imports.getHttpResponseBodyPtr = func() buffer.IoBuffer { return buffer.NewIoBufferString("resp body") }

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().Return(&AbiContext{imports: imports})

	assert.Equal(t, GetBuffer(instance, BufferTypeHttpRequestBody).String(), "req body")
	assert.Equal(t, GetBuffer(instance, BufferTypeHttpResponseBody).String(), "resp body")
	assert.Equal(t, GetBuffer(instance, BufferTypeVmConfiguration).String(), "vm config")
	assert.Equal(t, GetBuffer(instance, BufferTypePluginConfiguration).String(), "plugin config")
}

func TestGetMap(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	reqHeader := header.CommonHeader(map[string]string{})
	reqTrailer := header.CommonHeader(map[string]string{})
	respHeader := header.CommonHeader(map[string]string{})
	respTrailer := header.CommonHeader(map[string]string{})

	imports := newMockImportsHandler()
	imports.getHttpRequestHeaderPtr = func() api.HeaderMap { return reqHeader }
	imports.getHttpRequestTrailerPtr = func() api.HeaderMap { return reqTrailer }
	imports.getHttpResponseHeaderPtr = func() api.HeaderMap { return respHeader }
	imports.getHttpResponseTrailerPtr = func() api.HeaderMap { return respTrailer }

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().Return(&AbiContext{imports: imports})

	assert.Equal(t, GetMap(instance, MapTypeHttpRequestHeaders), reqHeader)
	assert.Equal(t, GetMap(instance, MapTypeHttpRequestTrailers), reqTrailer)
	assert.Equal(t, GetMap(instance, MapTypeHttpResponseHeaders), respHeader)
	assert.Equal(t, GetMap(instance, MapTypeHttpResponseTrailers), respTrailer)
}

func TestProxyLog(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().DoAndReturn(func() interface{} {
		imports := newMockImportsHandler()
		imports.logPtr = func(level log.Level, msg string) {
			assert.Equal(t, msg, "test log")
		}
		return &AbiContext{imports: imports}
	})
	instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).AnyTimes().Return([]byte("test log"), nil)

	assert.Equal(t, proxyLog(instance, int32(LogLevelInfo), int32(10), 8), WasmResultOk.Int32())
}

func TestProxyGetBufferBytes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().DoAndReturn(func() interface{} {
		imports := newMockImportsHandler()
		imports.getHttpRequestBodyPtr = func() buffer.IoBuffer {
			return buffer.NewIoBufferString("test body")
		}
		return &AbiContext{imports: imports}
	})
	instance.EXPECT().Malloc(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	var returnContent []byte
	instance.EXPECT().PutMemory(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(addr uint64, size uint64, content []byte) error {
		returnContent = content
		return nil
	})

	var returnPtr, returnSize uint32
	gomock.InOrder(
		instance.EXPECT().PutUint32(gomock.Any(), gomock.Any()).DoAndReturn(func(addr uint64, value uint32) error {
			returnPtr = value
			return nil
		}),
		instance.EXPECT().PutUint32(gomock.Any(), gomock.Any()).DoAndReturn(func(addr uint64, value uint32) error {
			returnSize = value
			return nil
		}),
	)

	assert.Equal(t, proxyGetBufferBytes(instance, int32(BufferTypeHttpRequestBody),
		0, 999, 0, 0), WasmResultOk.Int32())
	assert.Equal(t, string(returnContent), string("test body"))
	assert.Equal(t, returnPtr, uint32(1))
	assert.Equal(t, returnSize, uint32(9))
}

func TestProxySetBufferBytes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	data := buffer.NewIoBufferString("test data")

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().DoAndReturn(func() interface{} {
		imports := newMockImportsHandler()
		imports.getHttpRequestBodyPtr = func() buffer.IoBuffer { return data }
		return &AbiContext{imports: imports}
	})
	instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return([]byte("aaaa"), nil)

	assert.Equal(t, proxySetBufferBytes(instance, int32(BufferTypeHttpRequestBody), 0, 4, 0, 0), WasmResultOk.Int32())
	assert.Equal(t, data.String(), "aaaa data")
}

func TestProxyGetHeaderMapPairs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := map[string]string{"a": "aa", "b": "bb"}

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().DoAndReturn(func() interface{} {
		imports := newMockImportsHandler()
		imports.getHttpRequestHeaderPtr = func() api.HeaderMap {
			return header.CommonHeader(m)
		}
		return &AbiContext{imports: imports}
	})
	instance.EXPECT().Malloc(gomock.Any()).AnyTimes().Return(uint64(1), nil)
	instance.EXPECT().PutByte(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	instance.EXPECT().PutMemory(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	uint32Arr := make([]uint32, 0)
	instance.EXPECT().PutUint32(gomock.Any(), gomock.Any()).AnyTimes().DoAndReturn(func(addr uint64, value uint32) error {
		uint32Arr = append(uint32Arr, value)
		return nil
	})

	encodedMap := EncodeMap(m)

	assert.Equal(t, proxyGetHeaderMapPairs(instance, int32(MapTypeHttpRequestHeaders), 0, 0), WasmResultOk.Int32())
	assert.Equal(t, uint32Arr[len(uint32Arr)-2], uint32(1))
	assert.Equal(t, uint32Arr[len(uint32Arr)-1], uint32(len(encodedMap)))
}

func TestProxySetHeaderMapPairs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := map[string]string{"a": "aa", "b": "bb"}
	delta := EncodeMap(map[string]string{"b": "bbb", "c": "cc"})

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().DoAndReturn(func() interface{} {
		imports := newMockImportsHandler()
		imports.getHttpRequestHeaderPtr = func() api.HeaderMap {
			return header.CommonHeader(m)
		}
		return &AbiContext{imports: imports}
	})
	instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return(delta, nil)

	assert.Equal(t, proxySetHeaderMapPairs(instance, int32(MapTypeHttpRequestHeaders), 0, 0), WasmResultOk.Int32())
	assert.Equal(t, m["b"], "bbb")
	assert.Equal(t, m["c"], "cc")
}

func TestProxyGetHeaderMapValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := map[string]string{"a": "aa", "b": "bb"}

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().DoAndReturn(func() interface{} {
		imports := newMockImportsHandler()
		imports.getHttpRequestHeaderPtr = func() api.HeaderMap {
			return header.CommonHeader(m)
		}
		return &AbiContext{imports: imports}
	})
	instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return([]byte("a"), nil)
	instance.EXPECT().Malloc(gomock.Any()).AnyTimes().Return(uint64(1), nil)

	var returnContent []byte
	instance.EXPECT().PutMemory(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(addr uint64, size uint64, content []byte) error {
		returnContent = content
		return nil
	})

	var returnPtr, returnSize uint32
	gomock.InOrder(
		instance.EXPECT().PutUint32(gomock.Any(), gomock.Any()).DoAndReturn(func(addr uint64, value uint32) error {
			returnPtr = value
			return nil
		}),
		instance.EXPECT().PutUint32(gomock.Any(), gomock.Any()).DoAndReturn(func(addr uint64, value uint32) error {
			returnSize = value
			return nil
		}),
	)

	assert.Equal(t, proxyGetHeaderMapValue(instance, int32(MapTypeHttpRequestHeaders), 0, 0, 0, 0), WasmResultOk.Int32())
	assert.Equal(t, string(returnContent), string("aa"))
	assert.Equal(t, returnPtr, uint32(1))
	assert.Equal(t, returnSize, uint32(2))
}

func TestProxySetHeaderMapValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := map[string]string{"a": "aa", "b": "bb"}

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().DoAndReturn(func() interface{} {
		imports := newMockImportsHandler()
		imports.getHttpRequestHeaderPtr = func() api.HeaderMap {
			return header.CommonHeader(m)
		}
		return &AbiContext{imports: imports}
	})
	gomock.InOrder(
		instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return([]byte("b"), nil),
		instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return([]byte("bbb"), nil),
	)

	assert.Equal(t, proxyReplaceHeaderMapValue(instance, int32(MapTypeHttpRequestHeaders), 0, 0, 0, 0), WasmResultOk.Int32())
	assert.Equal(t, m["b"], "bbb")
}

func TestProxyAddHeaderMapValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := map[string]string{"a": "aa", "b": "bb"}

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().DoAndReturn(func() interface{} {
		imports := newMockImportsHandler()
		imports.getHttpRequestHeaderPtr = func() api.HeaderMap {
			return header.CommonHeader(m)
		}
		return &AbiContext{imports: imports}
	})
	gomock.InOrder(
		instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return([]byte("c"), nil),
		instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return([]byte("cc"), nil),
	)

	assert.Equal(t, proxyAddHeaderMapValue(instance, int32(MapTypeHttpRequestHeaders), 0, 0, 0, 0), WasmResultOk.Int32())
	assert.Equal(t, m["c"], "cc")
}

func TestProxyDelHeaderMapValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	m := map[string]string{"a": "aa", "b": "bb"}

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().DoAndReturn(func() interface{} {
		imports := newMockImportsHandler()
		imports.getHttpRequestHeaderPtr = func() api.HeaderMap {
			return header.CommonHeader(m)
		}
		return &AbiContext{imports: imports}
	})

	instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return([]byte("b"), nil)

	assert.Equal(t, proxyRemoveHeaderMapValue(instance, int32(MapTypeHttpRequestHeaders), 0, 0), WasmResultOk.Int32())
	assert.Equal(t, len(m), 1)
	_, ok := m["b"]
	assert.False(t, ok)
}

func TestProxyUnimplemented(t *testing.T) {
	assert.Equal(t, proxyGetProperty(nil, 0, 0, 0, 0), WasmResultOk.Int32())
	assert.Equal(t, proxySetProperty(nil, 0, 0, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxySetEffectiveContext(nil, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxySetTickPeriodMilliseconds(nil, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyGetCurrentTimeNanoseconds(nil, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyGrpcCall(nil, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyGrpcStream(nil, 0, 0, 0, 0, 0, 0, 0, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyGrpcCancel(nil, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyGrpcClose(nil, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyGrpcSend(nil, 0, 0, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxySetSharedData(nil, 0, 0, 0, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyGetSharedData(nil, 0, 0, 0, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyEnqueueSharedQueue(nil, 0, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyDequeueSharedQueue(nil, 0, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyResolveSharedQueue(nil, 0, 0, 0, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyGetMetric(nil, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyRegisterSharedQueue(nil, 0, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyRecordMetric(nil, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyIncrementMetric(nil, 0, 0), WasmResultUnimplemented.Int32())
	assert.Equal(t, proxyDefineMetric(nil, 0, 0, 0, 0), WasmResultUnimplemented.Int32())
}

func TestProxyHttpCall(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	server := http.Server{Addr: "127.0.0.1:22164"}
	defer server.Close()

	go func() {
		http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte("response from external server"))
		})
		server.ListenAndServe()
	}()
	time.Sleep(time.Second)

	instance := mock.NewMockWasmInstance(ctrl)
	instance.EXPECT().GetData().AnyTimes().Return(&AbiContext{instance: instance, imports: newMockImportsHandler()})
	gomock.InOrder(
		instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return([]byte("http://127.0.0.1:22164/"), nil),
		instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return(EncodeMap(map[string]string{"h1": "11", "h2": "22", "h3": "33"}), nil),
		instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return([]byte("req body"), nil),
		instance.EXPECT().GetMemory(gomock.Any(), gomock.Any()).Return(EncodeMap(map[string]string{"h4": "44"}), nil),
	)
	instance.EXPECT().PutUint32(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)
	f := mock.NewMockWasmFunction(ctrl)
	callHttpCallResp := false
	f.EXPECT().Call(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(interface{}, interface{}, interface{}, interface{}, interface{}) (interface{}, error) {
			callHttpCallResp = true
			return nil, nil
		})
	instance.EXPECT().GetExportsFunc(gomock.Any()).AnyTimes().Return(f, nil)

	assert.Equal(t, proxyHttpCall(instance, 0, 0, 0, 1, 0, 1, 0, 1, 50000, 0), WasmResultOk.Int32())
	time.Sleep(time.Second)
	assert.True(t, callHttpCallResp)
}
