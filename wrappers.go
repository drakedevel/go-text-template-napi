package main

// #include <node_api.h>
// #include "wrappers.inc"
import "C"
import "unsafe"

type napiEnv struct {
	inner C.napi_env
}

func (env napiEnv) CreateFunction(name string, data unsafe.Pointer, cb C.napi_callback) C.napi_value {
	var result C.napi_value
	status := C.NapiCreateFunction(env.inner, name, cb, data, &result)
	if status != C.napi_ok {
		panic(status)
	}
	return result
}

func (env napiEnv) CreateString(str string) C.napi_value {
	var result C.napi_value
	status := C.NapiCreateString(env.inner, str, &result)
	if status != C.napi_ok {
		panic(status)
	}
	return result
}

func (env napiEnv) DefineClass(name string, constructor C.napi_callback, data unsafe.Pointer, properties []C.napi_property_descriptor) C.napi_value {
	var result C.napi_value
	status := C.NapiDefineClass(env.inner, "Template", constructor, data, C.size_t(len(properties)), &properties[0], &result)
	if status != C.napi_ok {
		panic(status)
	}
	return result
}
