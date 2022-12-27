package napi

// #include <node_api.h>
import "C"

import "unsafe"

type PropertyAttributes C.napi_property_attributes

const (
	Default           PropertyAttributes = C.napi_default
	Writable          PropertyAttributes = C.napi_writable
	Enumerable        PropertyAttributes = C.napi_enumerable
	Configurable      PropertyAttributes = C.napi_configurable
	Static            PropertyAttributes = C.napi_static
	DefaultMethod     PropertyAttributes = C.napi_default_method
	DefaultJsProperty PropertyAttributes = C.napi_default_jsproperty
)

type PropertyDescriptor struct {
	Name Value

	Method Callback
	Getter Callback
	Setter Callback
	Value  Value

	Attributes PropertyAttributes
	Data       unsafe.Pointer
}

func (pd *PropertyDescriptor) toNative(env Env) C.napi_property_descriptor {
	return C.napi_property_descriptor{
		name: C.napi_value(pd.Name),

		method: C.napi_callback(pd.Method),
		getter: C.napi_callback(pd.Getter),
		setter: C.napi_callback(pd.Setter),
		value:  C.napi_value(pd.Value),

		attributes: C.napi_property_attributes(pd.Attributes),
		data:       pd.Data,
	}
}
