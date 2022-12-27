package main

// #include <stdlib.h>
// #include <node_api.h>
// napi_value genericNapiCallback(napi_env env, napi_callback_info info);
// void genericNapiFinalize(napi_env env, void* data, void* hint);
import "C"
import (
	"runtime/cgo"
	"unsafe"
)

type napiCallback func(env napiEnv, value C.napi_callback_info) C.napi_value
type napiFinalize func(env napiEnv, data unsafe.Pointer)
type cleanupFunc func()

//export genericNapiCallback
func genericNapiCallback(env C.napi_env, info C.napi_callback_info) C.napi_value {
	var data unsafe.Pointer
	status := C.napi_get_cb_info(env, info, nil, nil, nil, &data)
	if status != C.napi_ok {
		panic(status)
	}
	return unlaunderHandle(data).Value().(napiCallback)(napiEnv{env}, info)
}

func makeNapiCallback(cb napiCallback) (C.napi_callback, unsafe.Pointer, cleanupFunc) {
	ptr, cleanup := launderHandle(cgo.NewHandle(cb))
	return C.napi_callback(C.genericNapiCallback), ptr, cleanup
}

//export genericNapiFinalize
func genericNapiFinalize(env C.napi_env, data unsafe.Pointer, hint unsafe.Pointer) {
	unlaunderHandle(hint).Value().(napiFinalize)(napiEnv{env}, data)
}

func makeNapiFinalize(cb napiFinalize) (C.napi_finalize, unsafe.Pointer, cleanupFunc) {
	ptr, cleanup := launderHandle(cgo.NewHandle(cb))
	return C.napi_finalize(C.genericNapiFinalize), ptr, cleanup
}

func launderHandle(handle cgo.Handle) (unsafe.Pointer, cleanupFunc) {
	result := unsafe.Pointer(C.malloc(C.size_t(unsafe.Sizeof(uintptr(0)))))
	*((*uintptr)(result)) = uintptr(handle)
	return result, func() { deleteLaunderedHandle(result) }
}

func unlaunderHandle(ptr unsafe.Pointer) cgo.Handle {
	return cgo.Handle(*(*uintptr)(ptr))
}

func deleteLaunderedHandle(ptr unsafe.Pointer) {
	handle := unlaunderHandle(ptr)
	handle.Delete()
	C.free(ptr)
}
