package napi

// #cgo linux LDFLAGS: -Wl,--unresolved-symbols=ignore-all
// #cgo darwin LDFLAGS: -Wl,-undefined,dynamic_lookup
// #include <node_api.h>
// napi_value napiModuleInit(napi_env env, napi_value exports);
// NAPI_MODULE(go_text_template_napi_binding, napiModuleInit)
import "C"
