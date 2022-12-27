package napi

// #include <stdbool.h>
// #include <stdint.h>
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

func (env Env) GetArrayLength(value Value) uint32 {
	var result C.uint32_t
	status := C.napi_get_array_length(env.inner, C.napi_value(value), &result)
	if status != C.napi_ok {
		panic(status)
	}
	return uint32(result)
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

// Working with JavaScript values and abstract operations

type ValueType C.napi_valuetype

const (
	Undefined ValueType = C.napi_undefined
	Null      ValueType = C.napi_null
	Boolean   ValueType = C.napi_boolean
	Number    ValueType = C.napi_number
	String    ValueType = C.napi_string
	Symbol    ValueType = C.napi_symbol
	Object    ValueType = C.napi_object
	Function  ValueType = C.napi_function
	External  ValueType = C.napi_external
	Bigint    ValueType = C.napi_bigint
)

func (env Env) Typeof(value Value) ValueType {
	var result C.napi_valuetype
	status := C.napi_typeof(env.inner, C.napi_value(value), &result)
	if status != C.napi_ok {
		panic(status)
	}
	return ValueType(result)
}

func (env Env) IsArray(value Value) bool {
	var result C.bool
	status := C.napi_is_array(env.inner, C.napi_value(value), &result)
	if status != C.napi_ok {
		panic(status)
	}
	return bool(result)
}

// Working with JavaScript properties

type KeyCollectionMode C.napi_key_collection_mode
type KeyFilter C.napi_key_filter
type KeyConversion C.napi_key_conversion

const (
	KeyIncludePrototypes KeyCollectionMode = C.napi_key_include_prototypes
	KeyOwnOnly           KeyCollectionMode = C.napi_key_own_only

	KeyAllProperties KeyFilter = C.napi_key_all_properties
	KeyWritable      KeyFilter = C.napi_key_writable
	KeyEnumerable    KeyFilter = C.napi_key_enumerable
	KeyConfigurable  KeyFilter = C.napi_key_configurable
	KeySkipStrings   KeyFilter = C.napi_key_skip_strings
	KeySkipSymbols   KeyFilter = C.napi_key_skip_symbols

	KeyKeepNumbers      KeyConversion = C.napi_key_keep_numbers
	KeyNumbersToStrings KeyConversion = C.napi_key_numbers_to_strings
)

func (env Env) GetAllPropertyNames(object Value, keyMode KeyCollectionMode, keyFilter KeyFilter, keyConversion KeyConversion) Value {
	var result C.napi_value
	status := C.napi_get_all_property_names(
		env.inner,
		C.napi_value(object),
		C.napi_key_collection_mode(keyMode),
		C.napi_key_filter(keyFilter),
		C.napi_key_conversion(keyConversion),
		&result,
	)
	if status != C.napi_ok {
		panic(status)
	}
	return Value(result)
}

func (env Env) GetProperty(object Value, key Value) Value {
	var result C.napi_value
	status := C.napi_get_property(env.inner, C.napi_value(object), C.napi_value(key), &result)
	if status != C.napi_ok {
		panic(status)
	}
	return Value(result)
}

func (env Env) GetElement(object Value, index uint32) Value {
	var result C.napi_value
	status := C.napi_get_element(env.inner, C.napi_value(object), C.uint32_t(index), &result)
	if status != C.napi_ok {
		panic(status)
	}
	return Value(result)
}

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
