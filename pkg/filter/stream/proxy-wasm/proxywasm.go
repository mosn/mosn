package proxywasm

// #include <stdlib.h>
//
// extern int proxy_log(void *context, int, int, int);
// extern int proxy_get_property(void *context, int, int, int, int);
// extern int proxy_get_header_map_pairs(void *context, int, int, int);
// extern int proxy_get_buffer_bytes(void *context, int, int, int, int, int);
// extern int proxy_replace_header_map_value(void *context, int, int, int, int, int);
// extern int proxy_add_header_map_value(void *context, int, int, int, int, int);
import "C"

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	wasm "github.com/wasmerio/go-ext-wasm/wasmer"
	"mosn.io/api"
	"mosn.io/mosn/pkg/log"
)

type MapType int32

const (
	MapTypeHttpRequestHeaders       MapType = 0
	MapTypeHttpRequestTrailers      MapType = 1
	MapTypeHttpResponseHeaders      MapType = 2
	MapTypeHttpResponseTrailers     MapType = 3
	MapTypeHttpCallResponseHeaders  MapType = 7
	MapTypeHttpCallResponseTrailers MapType = 8
)

type BufferType int32

const (
	BufferTypeHttpRequestBody      BufferType = 0
	BufferTypeHttpResponseBody     BufferType = 1
	BufferTypeDownstreamData       BufferType = 2
	BufferTypeUpstreamData         BufferType = 3
	BufferTypeHttpCallResponseBody BufferType = 4
)

//export proxy_get_buffer_bytes
func proxy_get_buffer_bytes(context unsafe.Pointer, bufferType int32, start int32, maxSize int32, returnBufferData int32, returnBufferSize int32) int32 {
	var instanceContext = wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*wasmContext)

	var body []byte
	switch BufferType(bufferType) {
	case BufferTypeHttpRequestBody:
		body = ctx.filter.rhandler.GetRequestData().Bytes()
	case BufferTypeHttpResponseBody:
		body = ctx.filter.shandler.GetResponseData().Bytes()
	}

	r, _ := ctx.instance.Exports["malloc"](len(body))
	p := r.ToI32()
	memory := ctx.instance.Memory.Data()
	copy(memory[p:], body)

	binary.LittleEndian.PutUint32(memory[returnBufferData:], uint32(p))
	binary.LittleEndian.PutUint32(memory[returnBufferSize:], uint32(len(body)))

	return 0
}

//export proxy_get_header_map_pairs
func proxy_get_header_map_pairs(context unsafe.Pointer, mapType int32, keyData int32, keySize int32) int32 {
	var instanceContext = wasm.IntoInstanceContext(context)
	ctx := instanceContext.Data().(*wasmContext)

	var header api.HeaderMap
	switch MapType(mapType) {
	case MapTypeHttpRequestHeaders:
		header = ctx.filter.rhandler.GetRequestHeaders()
	case MapTypeHttpResponseHeaders:
		header = ctx.filter.shandler.GetResponseHeaders()
	}

	headerSize := 0

	f := ctx.instance.Exports["malloc"]
	r, err := f(int32(header.ByteSize() + 200))
	if err != nil {
		fmt.Printf("wasm malloc error: %v", err)
	}
	start := r.ToI32()
	memory := ctx.instance.Memory.Data()
	p := start

	p += 4
	header.Range(func(key string, value string) bool {
		headerSize++
		binary.LittleEndian.PutUint32(memory[p:], uint32(len(key)))
		p += 4
		binary.LittleEndian.PutUint32(memory[p:], uint32(len(value)))
		p += 4
		return true
	})

	header.Range(func(key string, value string) bool {
		copy(memory[p:], key)
		p += int32(len(key))
		memory[p] = 0
		p++

		copy(memory[p:], value)
		p += int32(len(value))
		memory[p] = 0
		p++

		return true
	})

	binary.LittleEndian.PutUint32(memory[start:], uint32(headerSize))

	binary.LittleEndian.PutUint32(memory[keyData:], uint32(start))
	binary.LittleEndian.PutUint32(memory[keySize:], uint32(p-start))

	return 0
}

//export proxy_replace_header_map_value
func proxy_replace_header_map_value(context unsafe.Pointer, mapType int32, keyData int32, keySize int32, valueData int32, valueSize int32) int32 {
	var instanceContext = wasm.IntoInstanceContext(context)
	var memory = instanceContext.Memory().Data()

	ctx := instanceContext.Data().(*wasmContext)

	key := string(memory[keyData : keyData+keySize])
	value := string(memory[valueData : valueData+valueSize])

	var header api.HeaderMap
	switch MapType(mapType) {
	case MapTypeHttpRequestHeaders:
		header = ctx.filter.rhandler.GetRequestHeaders()
		header.Set(key, value)
	case MapTypeHttpResponseHeaders:
		header = ctx.filter.shandler.GetResponseHeaders()
		header.Set(key, value)
	}

	return 0
}

//export proxy_add_header_map_value
func proxy_add_header_map_value(context unsafe.Pointer, mapType int32, keyData int32, keySize int32, valueData int32, valueSize int32) int32 {
	var instanceContext = wasm.IntoInstanceContext(context)
	var memory = instanceContext.Memory().Data()

	ctx := instanceContext.Data().(*wasmContext)

	key := string(memory[keyData : keyData+keySize])
	value := string(memory[valueData : valueData+valueSize])

	var header api.HeaderMap
	switch MapType(mapType) {
	case MapTypeHttpRequestHeaders:
		header = ctx.filter.rhandler.GetRequestHeaders()
		header.Add(key, value)
	case MapTypeHttpResponseHeaders:
		header = ctx.filter.shandler.GetResponseHeaders()
		header.Add(key, value)
	}

	return 0
}

//export proxy_log
func proxy_log(context unsafe.Pointer, logLevel int32, messageData int32, messageSize int32) int32 {
	var instanceContext = wasm.IntoInstanceContext(context)
	var memory = instanceContext.Memory().Data()

	msg := string(memory[messageData : messageData+messageSize])
	log.DefaultLogger.Errorf("wasm log: %s", msg)
	return 0
}

//export proxy_get_property
func proxy_get_property(context unsafe.Pointer, pathData int32, pathSize int32, returnValueData int32, returnValueSize int32) int32 {
	id := "my_root_id"

	var instanceContext = wasm.IntoInstanceContext(context)

	instance := instanceContext.Data().(*wasmContext).instance
	f := instance.Exports["malloc"]
	r, _ := f(len(id))
	p := r.ToI32()
	memory := instance.Memory.Data()
	copy(memory[p:], id)

	binary.LittleEndian.PutUint32(memory[returnValueData:], uint32(p))
	binary.LittleEndian.PutUint32(memory[returnValueSize:], uint32(len(id)))

	return 0
}

var root_id = 100

type wasmContext struct {
	filter   *streamProxyWasmFilter
	instance *wasm.Instance
}

func initWasm(path string) *wasm.Instance {
	bytes, _ := wasm.ReadBytes(path)
	module, err := wasm.Compile(bytes)
	ver := wasm.WasiGetVersion(module)

	importObject := wasm.NewDefaultWasiImportObjectForVersion(ver)

	im := wasm.NewImports()
	im, _ = im.AppendFunction("proxy_log", proxy_log, C.proxy_log)
	im, _ = im.AppendFunction("proxy_get_property", proxy_get_property, C.proxy_get_property)
	im, _ = im.AppendFunction("proxy_get_header_map_pairs", proxy_get_header_map_pairs, C.proxy_get_header_map_pairs)
	im, _ = im.AppendFunction("proxy_get_buffer_bytes", proxy_get_buffer_bytes, C.proxy_get_buffer_bytes)
	im, _ = im.AppendFunction("proxy_replace_header_map_value", proxy_replace_header_map_value, C.proxy_replace_header_map_value)
	im, _ = im.AppendFunction("proxy_add_header_map_value", proxy_add_header_map_value, C.proxy_add_header_map_value)

	err = importObject.Extend(*im)

	instance, err := module.InstantiateWithImportObject(importObject)
	if err != nil {
		log.DefaultLogger.Errorf("wasm instance error :%v", err)
		return nil
	}

	if _, err := instance.Exports["_start"](); err != nil {
		log.DefaultLogger.Errorf("wasm start err: %v\n", err)
		return nil
	}

	instance.SetContextData(&wasmContext{
		nil, &instance,
	})

	if _, err := instance.Exports["proxy_on_context_create"](root_id, 0); err != nil {
		log.DefaultLogger.Errorf("root err %v\n", err)
		return nil
	}

	if _, err := instance.Exports["proxy_on_vm_start"](root_id, 1000); err != nil {
		log.DefaultLogger.Errorf("start err %v\n", err)
		return nil
	}

	return &instance
}
