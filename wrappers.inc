#include <node_api.h>

napi_status NapiCreateFunction(napi_env env,
                               _GoString_ name,
                               napi_callback cb,
                               void* data,
                               napi_value* result) {
  return napi_create_function(env, _GoStringPtr(name), _GoStringLen(name), cb,
                              data, result);
}

napi_status NapiCreateString(napi_env env, _GoString_ str, napi_value* result) {
  return napi_create_string_utf8(env, _GoStringPtr(str), _GoStringLen(str),
                                 result);
}

napi_status NapiDefineClass(napi_env env,
                            _GoString_ name,
                            napi_callback constructor,
                            void* data,
                            size_t property_count,
                            const napi_property_descriptor* properties,
                            napi_value* result) {
  return napi_define_class(env, _GoStringPtr(name), _GoStringLen(name),
                           constructor, data, property_count, properties,
                           result);
}
