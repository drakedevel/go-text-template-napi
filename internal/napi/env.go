package napi

// #include <stdbool.h>
// #include <stdint.h>
// #include <stdlib.h>
// #include <node_api.h>
// #include "env_prelude.inc"
import "C"
import (
	"fmt"
	"unsafe"
)

type Env struct {
	inner C.napi_env
}

// Error handling

type ExtendedErrorInfo struct {
	errorMessage    string
	engineReserved  unsafe.Pointer
	engineErrorCode uint32
	errorCode       C.napi_status
}

func (env Env) GetLastErrorInfo() (*ExtendedErrorInfo, error) {
	var info *C.napi_extended_error_info
	status := C.napi_get_last_error_info(env.inner, &info)
	if status != C.napi_ok {
		return nil, fmt.Errorf("failed to get N-API error info: %d", status)
	}
	return &ExtendedErrorInfo{
		errorMessage:    C.GoString(info.error_message),
		engineReserved:  info.engine_reserved,
		engineErrorCode: uint32(info.engine_error_code),
		errorCode:       info.error_code,
	}, nil
}

func (env Env) ThrowError(code string, msg string) error {
	var cCode *C.char
	if code != "" {
		cCode := C.CString(code)
		defer C.free(unsafe.Pointer(cCode))
	}
	cMsg := C.CString(msg)
	defer C.free(unsafe.Pointer(cMsg))
	return env.mapStatus(C.napi_throw_error(env.inner, cCode, cMsg))
}

func (env Env) ThrowTypeError(code string, msg string) error {
	var cCode *C.char
	if code != "" {
		cCode = C.CString(code)
		defer C.free(unsafe.Pointer(cCode))
	}
	cMsg := C.CString(msg)
	defer C.free(unsafe.Pointer(cMsg))
	return env.mapStatus(C.napi_throw_type_error(env.inner, cCode, cMsg))
}

func (env Env) IsExceptionPending() (bool, error) {
	var result C.bool
	status := C.napi_is_exception_pending(env.inner, &result)
	if err := env.mapStatus(status); err != nil {
		return false, err
	}
	return bool(result), nil
}

func (env Env) FatalException(errValue Value) error {
	return env.mapStatus(C.napi_fatal_exception(env.inner, C.napi_value(errValue)))
}

func (env Env) mapStatus(status C.napi_status) error {
	if status == C.napi_ok {
		return nil
	}
	info, err := env.GetLastErrorInfo()
	if err != nil {
		return fmt.Errorf("N-API error code %v. Error getting extended info: %w", status, err)
	}
	return fmt.Errorf("N-API Error: %s (code %d)", info.errorMessage, info.errorCode)
}

func (env Env) maybeThrowError(err error) {
	// Don't clobber a pending exception if there is one
	isPending, pendErr := env.IsExceptionPending()
	if pendErr != nil || isPending {
		return
	}

	// TODO: Add mechanism for throwing custom errors here
	// (Perhaps with errors.As on a napiThrow interface?)
	throwErr := env.ThrowError("", err.Error())
	if throwErr != nil {
		// TODO: Anything more useful to do here?
		fmt.Println("N-API error", throwErr, "throwing error", err)
	}
}

// Working with JavaScript values

func (env Env) CreateFunction(name string, data unsafe.Pointer, cb Callback) (Value, error) {
	var result C.napi_value
	status := C.NapiCreateFunction(env.inner, name, cb, data, &result)
	if err := env.mapStatus(status); err != nil {
		return nil, err
	}
	return Value(result), nil
}

func (env Env) CreateString(str string) (Value, error) {
	var result C.napi_value
	status := C.NapiCreateString(env.inner, str, &result)
	if err := env.mapStatus(status); err != nil {
		return nil, err
	}
	return Value(result), nil
}

func (env Env) GetArrayLength(value Value) (uint32, error) {
	var result C.uint32_t
	status := C.napi_get_array_length(env.inner, C.napi_value(value), &result)
	if err := env.mapStatus(status); err != nil {
		return 0, err
	}
	return uint32(result), nil
}

func (env Env) GetValueBool(value Value) (bool, error) {
	var result C.bool
	status := C.napi_get_value_bool(env.inner, C.napi_value(value), &result)
	if err := env.mapStatus(status); err != nil {
		return false, err
	}
	return bool(result), nil
}

func (env Env) GetValueDouble(value Value) (float64, error) {
	var result C.double
	status := C.napi_get_value_double(env.inner, C.napi_value(value), &result)
	if err := env.mapStatus(status); err != nil {
		return 0, err
	}
	return float64(result), nil
}

func (env Env) GetValueBigintWords(value Value, signBit *int, wordCount *int, words *uint64) error {
	var cSignBit *C.int
	if signBit != nil {
		cSignBit = new(C.int)
	}
	var cWordCount C.size_t
	if wordCount != nil {
		cWordCount = C.size_t(*wordCount)
	}
	status := C.napi_get_value_bigint_words(env.inner, C.napi_value(value), cSignBit, &cWordCount, (*C.uint64_t)(words))
	if err := env.mapStatus(status); err != nil {
		return err
	}
	if signBit != nil {
		*signBit = int(*cSignBit)
	}
	if wordCount != nil {
		*wordCount = int(cWordCount)
	}
	return nil
}

func (env Env) GetValueString(value Value, buf []byte) (int, error) {
	// TODO: Maybe go ahead and handle allocating the buffer / converting to string?
	var result C.size_t
	var bufPtr *C.char
	if len(buf) > 0 {
		bufPtr = (*C.char)(unsafe.Pointer(&buf[0]))
	}

	status := C.napi_get_value_string_utf8(env.inner, C.napi_value(value), bufPtr, C.size_t(len(buf)), &result)
	if err := env.mapStatus(status); err != nil {
		return 0, err
	}
	return int(result), nil
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

func (env Env) Typeof(value Value) (ValueType, error) {
	var result C.napi_valuetype
	status := C.napi_typeof(env.inner, C.napi_value(value), &result)
	if err := env.mapStatus(status); err != nil {
		return 0, err
	}
	return ValueType(result), nil
}

func (env Env) IsArray(value Value) (bool, error) {
	var result C.bool
	status := C.napi_is_array(env.inner, C.napi_value(value), &result)
	if err := env.mapStatus(status); err != nil {
		return false, err
	}
	return bool(result), nil
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

func (env Env) GetAllPropertyNames(object Value, keyMode KeyCollectionMode, keyFilter KeyFilter, keyConversion KeyConversion) (Value, error) {
	var result C.napi_value
	status := C.napi_get_all_property_names(
		env.inner,
		C.napi_value(object),
		C.napi_key_collection_mode(keyMode),
		C.napi_key_filter(keyFilter),
		C.napi_key_conversion(keyConversion),
		&result,
	)
	if err := env.mapStatus(status); err != nil {
		return nil, err
	}
	return Value(result), nil
}

func (env Env) GetProperty(object Value, key Value) (Value, error) {
	var result C.napi_value
	status := C.napi_get_property(env.inner, C.napi_value(object), C.napi_value(key), &result)
	if err := env.mapStatus(status); err != nil {
		return nil, err
	}
	return Value(result), nil
}

func (env Env) GetElement(object Value, index uint32) (Value, error) {
	var result C.napi_value
	status := C.napi_get_element(env.inner, C.napi_value(object), C.uint32_t(index), &result)
	if err := env.mapStatus(status); err != nil {
		return nil, err
	}
	return Value(result), nil
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

func (env Env) DefineProperties(object Value, properties []PropertyDescriptor) error {
	pdLen, pdPtr := env.convertPropertyDescriptors(properties)
	status := C.napi_define_properties(env.inner, object, pdLen, pdPtr)
	return env.mapStatus(status)
}

// Working with JavaScript functions

func (env Env) GetCbInfo(cbinfo CallbackInfo, argc *int, argv *Value, thisArg *Value, data *unsafe.Pointer) error {
	// TODO: Consider interface that returns, instead of outparams
	var nativeArgc C.size_t
	if argc != nil {
		nativeArgc = C.size_t(*argc)
	}

	status := C.napi_get_cb_info(env.inner, C.napi_callback_info(cbinfo), &nativeArgc, (*C.napi_value)(argv), (*C.napi_value)(thisArg), data)
	if err := env.mapStatus(status); err != nil {
		return err
	}
	if argc != nil {
		*argc = int(nativeArgc)
	}
	return nil
}

// Object wrap

func (env Env) DefineClass(name string, constructor Callback, data unsafe.Pointer, properties []PropertyDescriptor) (Value, error) {
	pdLen, pdPtr := env.convertPropertyDescriptors(properties)
	var result C.napi_value
	status := C.NapiDefineClass(env.inner, "Template", constructor, data, pdLen, pdPtr, &result)
	if err := env.mapStatus(status); err != nil {
		return nil, err
	}
	return Value(result), nil
}

func (env Env) Wrap(jsObject Value, nativeObject unsafe.Pointer, finalizeCb Finalize, finalizeHint unsafe.Pointer) error {
	// TODO: Optionally return result?
	status := C.napi_wrap(env.inner, C.napi_value(jsObject), nativeObject, C.napi_finalize(finalizeCb), finalizeHint, nil)
	return env.mapStatus(status)
}

func (env Env) Unwrap(jsObject Value) (unsafe.Pointer, error) {
	var result unsafe.Pointer
	status := C.napi_unwrap(env.inner, C.napi_value(jsObject), &result)
	if err := env.mapStatus(status); err != nil {
		return nil, err
	}
	return result, nil
}

type TypeTag C.napi_type_tag

func (env Env) TypeTagObject(jsObject Value, typeTag *TypeTag) error {
	status := C.napi_type_tag_object(env.inner, C.napi_value(jsObject), (*C.napi_type_tag)(typeTag))
	return env.mapStatus(status)
}

func (env Env) CheckObjectTypeTag(jsObject Value, typeTag *TypeTag) (bool, error) {
	var result C.bool
	status := C.napi_check_object_type_tag(env.inner, C.napi_value(jsObject), (*C.napi_type_tag)(typeTag), &result)
	if err := env.mapStatus(status); err != nil {
		return false, err
	}
	return bool(result), nil
}
