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
type finalizeFunc func(env Env, data unsafe.Pointer) error
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
	err := unlaunderHandle(hint).Value().(finalizeFunc)(env, data)
	if err != nil {
		// N-API won't propagate an exception from a finalizer, so just log it
		fmt.Println("Uncaught exception in finalizer:", err)
	}
}

// TODO: Re-evaluate this API
func MakeNapiFinalize(cb finalizeFunc) (Finalize, unsafe.Pointer, cleanupFunc) {
	ptr, cleanup := launderHandle(cgo.NewHandle(cb))
	return Finalize(C.genericNapiFinalize), ptr, cleanup
}

func makeDataAndFinalize(data interface{}) (unsafe.Pointer, Finalize, unsafe.Pointer) {
	handle := cgo.NewHandle(data)
	dataPtr, dataCleanup := launderHandle(handle)
	var finalizeCleanup cleanupFunc
	finalizeCb, finalizePtr, finalizeCleanup := MakeNapiFinalize(func(env Env, data unsafe.Pointer) error {
		dataCleanup()
		finalizeCleanup()
		return nil
	})
	return dataPtr, finalizeCb, finalizePtr

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
