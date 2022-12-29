package main

import (
	"fmt"
	"reflect"

	"github.com/drakedevel/go-text-template-napi/internal/napi"
)

func jsValueToGo(env napi.Env, value napi.Value) (interface{}, error) {
	valueType, err := env.Typeof(value)
	if err != nil {
		return nil, err
	}
	switch valueType {
	case napi.Undefined, napi.Null:
		// TODO: Filter out Undefined from parent object, keep Null
		return nil, nil
	case napi.Boolean:
		return env.GetValueBool(value)
	case napi.Number:
		return env.GetValueDouble(value)
	case napi.String:
		return extractString(env, value)
	case napi.Object:
		isArray, err := env.IsArray(value)
		if err != nil {
			return nil, err
		}
		if isArray {
			length, err := env.GetArrayLength(value)
			if err != nil {
				return nil, err
			}
			result := make([]interface{}, length)
			for i := uint32(0); i < length; i++ {
				// TODO: Scope?
				elt, err := env.GetElement(value, i)
				if err != nil {
					return nil, err
				}
				eltConv, err := jsValueToGo(env, elt)
				if err != nil {
					return nil, err
				}
				result[i] = eltConv
			}
			return result, nil
		} else {
			// TODO: Should any other object types get special handling?
			propNames, err := env.GetAllPropertyNames(value, napi.KeyOwnOnly, napi.KeySkipSymbols, napi.KeyNumbersToStrings)
			if err != nil {
				return nil, err
			}
			length, err := env.GetArrayLength(propNames)
			if err != nil {
				return nil, err
			}
			result := map[string]interface{}{}
			for i := uint32(0); i < length; i++ {
				// TODO: Scope?
				key, err := env.GetElement(propNames, i)
				if err != nil {
					return nil, err
				}
				keyStr, err := extractString(env, key)
				if err != nil {
					return nil, err
				}
				elt, err := env.GetProperty(value, key)
				if err != nil {
					return nil, err
				}
				eltConv, err := jsValueToGo(env, elt)
				if err != nil {
					return nil, err
				}
				result[keyStr] = eltConv
			}
			return result, nil
		}
	case napi.Bigint:
		return extractBigint(env, value)
	default:
		// No useful way to map these to Go types
		// TODO: More useful error message?
		err = env.ThrowTypeError("ERR_INVALID_ARG_TYPE", "Unsupported value type")
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("threw exception")
	}
}

func goValueToJs(env napi.Env, value interface{}) (napi.Value, error) {
	reflectValue := reflect.ValueOf(value)
	switch reflectValue.Kind() {
	case reflect.Invalid:
		return env.GetNull()
	case reflect.Bool:
		return env.GetBoolean(reflectValue.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return env.CreateInt64(reflectValue.Int())
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return env.CreateUint32(uint32(reflectValue.Uint()))
	// TODO: case reflect.Uint, reflect.Uint64, reflect.Uintptr:
	case reflect.Float32, reflect.Float64:
		// FIXME: Why does NaN wind up getting mapped to null here?
		return env.CreateDouble(reflectValue.Float())
	case reflect.Array, reflect.Slice:
		arrayLen := reflectValue.Len()
		jsArray, err := env.CreateArrayWithLength(arrayLen)
		if err != nil {
			return nil, err
		}
		for i := 0; i < arrayLen; i++ {
			jsVal, err := goValueToJs(env, reflectValue.Index(i).Interface())
			if err != nil {
				return nil, err
			}
			if err := env.SetElement(jsArray, uint32(i), jsVal); err != nil {
				return nil, err
			}
		}
		return jsArray, nil
	case reflect.Map:
		jsObj, err := env.CreateObject()
		if err != nil {
			return nil, err
		}
		iter := reflectValue.MapRange()
		for iter.Next() {
			mapKey := iter.Key()
			mapValue := iter.Value()
			if mapKey.Kind() != reflect.String {
				return nil, fmt.Errorf("can't convert Go map key with type %s", mapKey.Type())
			}
			jsKey, err := env.CreateString(mapKey.String())
			if err != nil {
				return nil, err
			}
			jsValue, err := goValueToJs(env, mapValue.Interface())
			if err != nil {
				return nil, err
			}
			if err := env.SetProperty(jsObj, jsKey, jsValue); err != nil {
				return nil, err
			}
		}
		return jsObj, nil
	case reflect.String:
		return env.CreateString(reflectValue.String())
	case reflect.Complex64, reflect.Complex128, reflect.Chan, reflect.Func,
		reflect.Interface, reflect.Pointer, reflect.Struct, reflect.UnsafePointer:
		fallthrough
	default:
		return nil, fmt.Errorf("can't convert Go value of type %s", reflectValue.Type())
	}
}
