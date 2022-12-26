package main

// #cgo CFLAGS: '-DNODE_GYP_MODULE_NAME=binding' '-DUSING_UV_SHARED=1' '-DUSING_V8_SHARED=1' '-DV8_DEPRECATION_WARNINGS=1' '-DV8_DEPRECATION_WARNINGS' '-DV8_IMMINENT_DEPRECATION_WARNINGS' '-D_GLIBCXX_USE_CXX11_ABI=1' '-D_LARGEFILE_SOURCE' '-D_FILE_OFFSET_BITS=64' '-D__STDC_FORMAT_MACROS' '-DOPENSSL_NO_PINSHARED' '-DOPENSSL_THREADS' '-DBUILDING_NODE_EXTENSION' -I/home/adrake/.cache/node-gyp/16.19.0/include/node -I/home/adrake/.cache/node-gyp/16.19.0/src -I/home/adrake/.cache/node-gyp/16.19.0/deps/openssl/config -I/home/adrake/.cache/node-gyp/16.19.0/deps/openssl/openssl/include -I/home/adrake/.cache/node-gyp/16.19.0/deps/uv/include -I/home/adrake/.cache/node-gyp/16.19.0/deps/zlib -I/home/adrake/.cache/node-gyp/16.19.0/deps/v8/include
// #cgo LDFLAGS: -Wl,--unresolved-symbols=ignore-all
// #include <node_api.h>
// napi_value Init(napi_env env, napi_value exports);
// napi_value ExportedFunc(napi_env env, napi_callback_info info);
//
// NAPI_MODULE(binding, Init)
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
