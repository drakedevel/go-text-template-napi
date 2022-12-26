package main

// #cgo LDFLAGS: -Wl,--unresolved-symbols=ignore-all
// #include <node_api.h>
// napi_value Init(napi_env env, napi_value exports);
// napi_value ExportedFunc(napi_env env, napi_callback_info info);
//
// NAPI_MODULE(NODE_GYP_BINDING_NAME, Init)
//
// napi_status NapiCreateString(napi_env env, _GoString_ str, napi_value *result) {
//   return napi_create_string_utf8(env, _GoStringPtr(str), _GoStringLen(str), result);
// }
//
// napi_status NapiCreateFunction(napi_env env, _GoString_ name, napi_callback cb, void *data, napi_value *result) {
//   return napi_create_function(env, _GoStringPtr(name), _GoStringLen(name), cb, data, result);
// }
import "C"
import "fmt"

func RealInit(env C.napi_env, exports C.napi_value) C.napi_value {
	fmt.Printf("In N-API module Init\n")
	var fn C.napi_value
	status := C.NapiCreateFunction(env, "hello", C.napi_callback(C.ExportedFunc), nil, &fn)
	if status != C.napi_ok {
		panic(status)
	}

	var propName C.napi_value
	status = C.NapiCreateString(env, "hello", &propName)
	if status != C.napi_ok {
		panic(status)
	}

	propDesc := C.napi_property_descriptor{
		name:       propName,
		value:      fn,
		attributes: C.napi_enumerable,
	}
	status = C.napi_define_properties(env, exports, 1, &propDesc)

	return exports
}

func RealExportedFunc(env C.napi_env, info C.napi_callback_info) C.napi_value {
	var result C.napi_value
	var status = C.NapiCreateString(env, "Hello, world!", &result)
	if status != C.napi_ok {
		panic(status)
	}
	return result
}

func init() {
	fmt.Printf("In init function\n")
}

func main() {}
