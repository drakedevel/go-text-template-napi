package main

// #cgo linux LDFLAGS: -Wl,--unresolved-symbols=ignore-all
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
// #include <node_api.h>
// napi_value napiModuleInit(napi_env env, napi_value exports);
// NAPI_MODULE(go_text_template_napi_binding, napiModuleInit)
import "C"
import "fmt"

//export napiModuleInit
func napiModuleInit(rawEnv C.napi_env, exports C.napi_value) C.napi_value {
	fmt.Printf("In N-API module Init\n")

	var propDescs []C.napi_property_descriptor

	env := napiEnv{rawEnv}
	propDescs = append(propDescs, C.napi_property_descriptor{
		name:       env.CreateString("Template"),
		value:      buildTemplateClass(env),
		attributes: C.napi_enumerable,
	})

	status := C.napi_define_properties(env.inner, exports, C.ulong(len(propDescs)), &propDescs[0])
	if status != C.napi_ok {
		panic(status)
	}

	return exports
}

func main() {}
