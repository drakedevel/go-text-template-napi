package napi

// #cgo linux LDFLAGS: -Wl,--unresolved-symbols=ignore-all
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
// #include <node_api.h>
// napi_value napiModuleInit(napi_env env, napi_value exports);
// NAPI_MODULE(go_text_template_napi_binding, napiModuleInit)
import "C"

var registeredInitFunc func(env Env, exports Value) Value

//export napiModuleInit
func napiModuleInit(env C.napi_env, exports C.napi_value) C.napi_value {
	if registeredInitFunc == nil {
		// FIXME: Throw
		panic("Module init function not registered")
	}
	return C.napi_value(registeredInitFunc(Env{env}, Value(exports)))
}

func SetModuleInit(init func(env Env, exports Value) Value) {
	if registeredInitFunc != nil {
		panic("Module init function already registered")
	}
	registeredInitFunc = init
}
