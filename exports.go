package main

// #include <node_api.h>
import "C"

//export Init
func Init(env C.napi_env, exports C.napi_value) C.napi_value {
	return RealInit(env, exports)
}

//export ExportedFunc
func ExportedFunc(env C.napi_env, info C.napi_callback_info) C.napi_value {
	return RealExportedFunc(env, info)
}
