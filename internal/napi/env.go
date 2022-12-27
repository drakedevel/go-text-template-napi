package napi

// #include <node_api.h>
// #include "env_prelude.inc"
import "C"
import "unsafe"

type Env struct {
	inner C.napi_env
}

// Working with JavaScript values

func (env Env) CreateFunction(name string, data unsafe.Pointer, cb Callback) Value {
	var result C.napi_value
	// XXX: Why does this typecheck with cb directly?
	status := C.NapiCreateFunction(env.inner, name, cb, data, &result)
	if status != C.napi_ok {
		panic(status)
	}
	return Value(result)
}

func (env Env) CreateString(str string) Value {
	var result C.napi_value
	status := C.NapiCreateString(env.inner, str, &result)
	if status != C.napi_ok {
		panic(status)
	}
	return Value(result)
}

func (env Env) GetValueString(value Value, buf []byte) int {
	// TODO: Maybe go ahead and handle allocating the buffer / converting to string?
	var result C.size_t
	var bufPtr *C.char
	if len(buf) > 0 {
		bufPtr = (*C.char)(unsafe.Pointer(&buf[0]))
	}

	status := C.napi_get_value_string_utf8(env.inner, C.napi_value(value), bufPtr, C.size_t(len(buf)), &result)
	if status != C.napi_ok {
		panic(status)
	}
	return int(result)
}

// Working with JavaScript properties

func (env Env) convertPropertyDescriptors(properties []PropertyDescriptor) (C.size_t, *C.napi_property_descriptor) {
	if len(properties) == 0 {
		return 0, nil
	}
	propDescs := make([]C.napi_property_descriptor, len(properties))
	for i, v := range properties {
		propDescs[i] = v.toNative(env)
	}
	return C.size_t(len(propDescs)), &propDescs[0]
}

func (env Env) DefineProperties(object Value, properties []PropertyDescriptor) {
	pdLen, pdPtr := env.convertPropertyDescriptors(properties)
	status := C.napi_define_properties(env.inner, object, pdLen, pdPtr)
	if status != C.napi_ok {
		panic(status)
	}
}

// Working with JavaScript functions

func (env Env) GetCbInfo(cbinfo CallbackInfo, argc *int, argv *Value, thisArg *Value, data *unsafe.Pointer) {
	// TODO: Consider interface that returns, instead of outparams
	var nativeArgc C.size_t
	if argc != nil {
		nativeArgc = C.size_t(*argc)
	}

	status := C.napi_get_cb_info(env.inner, C.napi_callback_info(cbinfo), &nativeArgc, (*C.napi_value)(argv), (*C.napi_value)(thisArg), data)
	if status != C.napi_ok {
		panic(status)
	}
	if argc != nil {
		*argc = int(nativeArgc)
	}
}

// Object wrap

func (env Env) DefineClass(name string, constructor Callback, data unsafe.Pointer, properties []PropertyDescriptor) Value {
	pdLen, pdPtr := env.convertPropertyDescriptors(properties)
	var result C.napi_value
	status := C.NapiDefineClass(env.inner, "Template", constructor, data, pdLen, pdPtr, &result)
	if status != C.napi_ok {
		panic(status)
	}
	return Value(result)
}

func (env Env) Wrap(jsObject Value, nativeObject unsafe.Pointer, finalizeCb Finalize, finalizeHint unsafe.Pointer) {
	// TODO: Optionally return result?
	status := C.napi_wrap(env.inner, C.napi_value(jsObject), nativeObject, C.napi_finalize(finalizeCb), finalizeHint, nil)
	if status != C.napi_ok {
		panic(status)
	}
}

func (env Env) Unwrap(jsObject Value) unsafe.Pointer {
	var result unsafe.Pointer
	status := C.napi_unwrap(env.inner, C.napi_value(jsObject), &result)
	if status != C.napi_ok {
		panic(status)
	}
	return result
}
