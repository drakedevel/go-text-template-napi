package napi

// #include <stdlib.h>
// #include <node_api.h>
// napi_value genericNapiCallback(napi_env env, napi_callback_info info);
// void genericNapiFinalize(napi_env env, void* data, void* hint);
import "C"
import (
	"runtime/cgo"
	"unsafe"
)

type Callback C.napi_callback
type CallbackInfo C.napi_callback_info
type Finalize C.napi_finalize

type callbackFunc func(env Env, value CallbackInfo) Value
type finalizeFunc func(env Env, data unsafe.Pointer)
type cleanupFunc func()

//export genericNapiCallback
func genericNapiCallback(rawEnv C.napi_env, rawInfo C.napi_callback_info) C.napi_value {
	env := Env{rawEnv}
	info := CallbackInfo(rawInfo)
	var data unsafe.Pointer
	env.GetCbInfo(info, nil, nil, nil, &data)
	return C.napi_value(unlaunderHandle(data).Value().(callbackFunc)(env, info))
}

func MakeNapiCallback(cb callbackFunc) (Callback, unsafe.Pointer, cleanupFunc) {
	ptr, cleanup := launderHandle(cgo.NewHandle(cb))
	return Callback(C.genericNapiCallback), ptr, cleanup
}

//export genericNapiFinalize
func genericNapiFinalize(env C.napi_env, data unsafe.Pointer, hint unsafe.Pointer) {
	unlaunderHandle(hint).Value().(finalizeFunc)(Env{env}, data)
}

func MakeNapiFinalize(cb finalizeFunc) (Finalize, unsafe.Pointer, cleanupFunc) {
	ptr, cleanup := launderHandle(cgo.NewHandle(cb))
	return Finalize(C.genericNapiFinalize), ptr, cleanup
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
