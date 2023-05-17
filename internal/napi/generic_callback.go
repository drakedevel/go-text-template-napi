package napi

// #include <stdlib.h>
// #include <node_api.h>
// napi_value genericNapiCallback(napi_env env, napi_callback_info info);
// void genericNapiFinalize(napi_env env, void* data, void* hint);
import "C"
import (
	"fmt"
	"runtime/cgo"
	"unsafe"
)

type Callback C.napi_callback
type CallbackInfo C.napi_callback_info
type Finalize C.napi_finalize

type callbackFunc func(env Env, value CallbackInfo) (Value, error)
type finalizeFunc func(env Env, data interface{}) error
type cleanupFunc func()

//export genericNapiCallback
func genericNapiCallback(rawEnv C.napi_env, rawInfo C.napi_callback_info) C.napi_value {
	env := Env{rawEnv}
	info := CallbackInfo(rawInfo)
	var data unsafe.Pointer
	if err := env.GetCbInfo(info, nil, nil, nil, &data); err != nil {
		env.maybeThrowError(err)
		return nil
	}
	result, err := unlaunderHandle(data).Value().(callbackFunc)(env, info)
	if err != nil {
		env.maybeThrowError(err)
		return nil
	}
	return C.napi_value(result)
}

func MakeNapiCallback(cb callbackFunc) (Callback, unsafe.Pointer, cleanupFunc) {
	ptr, cleanup := launderHandle(cgo.NewHandle(cb))
	return Callback(C.genericNapiCallback), ptr, cleanup
}

//export genericNapiFinalize
func genericNapiFinalize(rawEnv C.napi_env, data unsafe.Pointer, hint unsafe.Pointer) {
	env := Env{rawEnv}
	dataHandle := unlaunderHandle(data)
	hintHandle := unlaunderHandle(hint)
	if err := hintHandle.Value().(finalizeFunc)(env, dataHandle.Value()); err != nil {
		// Node-API won't propagate an exception from a finalizer, so just log it
		fmt.Println("Uncaught exception in finalizer:", err)
	}
	deleteLaunderedHandle(data)
	deleteLaunderedHandle(hint)
}

func makeDataAndFinalize(data interface{}, finalize finalizeFunc) (unsafe.Pointer, Finalize, unsafe.Pointer) {
	dataPtr, _ := launderHandle(cgo.NewHandle(data))
	hintPtr, _ := launderHandle(cgo.NewHandle(finalize))
	return dataPtr, Finalize(C.genericNapiFinalize), hintPtr
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
