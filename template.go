package main

// #include <node_api.h>
import "C"
import (
	"fmt"
	"runtime/cgo"
	"unsafe"
)

type template struct {
	name string
}

func extractString(env napiEnv, value C.napi_value) string {
	// Get string length
	var strLen C.size_t
	status := C.napi_get_value_string_utf8(env.inner, value, nil, 0, &strLen)
	if status != C.napi_ok {
		panic(status)
	}

	// Allocate buffer and get string contents
	buf := make([]byte, strLen+1)
	status = C.napi_get_value_string_utf8(env.inner, value, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)), &strLen)
	if status != C.napi_ok {
		panic(status)
	}
	return string(buf[0:strLen])
}

func templateMethodEntry(env napiEnv, info C.napi_callback_info) (this *template) {
	var thisArg C.napi_value
	// status := C.napi_get_cb_info(env.inner, info, &argc, &argv[0], &thisArg, nil)
	status := C.napi_get_cb_info(env.inner, info, nil, nil, &thisArg, nil)
	if status != C.napi_ok {
		panic(status)
	}

	// Retrieve wrapped native object from JS object
	// TODO: Type tagging
	var wrapped unsafe.Pointer
	status = C.napi_unwrap(env.inner, thisArg, &wrapped)
	if status != C.napi_ok {
		panic(status)
	}
	handle := *(*cgo.Handle)(wrapped)

	return handle.Value().(*template)

}

func buildTemplateClass(env napiEnv) C.napi_value {
	// Build property descriptors
	var propDescs []C.napi_property_descriptor

	// TODO: Don't leak fooData
	fooCb, fooData, _ := makeNapiCallback(templateMethodFoo)
	propDescs = append(propDescs, C.napi_property_descriptor{
		name:       env.CreateString("foo"),
		method:     fooCb,
		attributes: C.napi_default_method,
		data:       fooData,
	})

	// TODO: Don't leak barData
	barCb, barData, _ := makeNapiCallback(templateMethodBar)
	propDescs = append(propDescs, C.napi_property_descriptor{
		name:       env.CreateString("bar"),
		method:     barCb,
		attributes: C.napi_default_method,
		data:       barData,
	})

	// Define class
	// TODO: Don't leak consData
	consCb, consData, _ := makeNapiCallback(templateConstructor)
	return env.DefineClass("Template", consCb, consData, propDescs)
}

func templateConstructor(env napiEnv, info C.napi_callback_info) C.napi_value {
	argc := C.size_t(1)
	argv := make([]C.napi_value, 1)
	var thisArg C.napi_value
	status := C.napi_get_cb_info(env.inner, info, &argc, &argv[0], &thisArg, nil)
	if status != C.napi_ok {
		panic(status)
	}
	if argc != 1 {
		// TODO: Throw
		panic("expected an argument")
	}

	data := template{name: extractString(env, argv[0])}
	fmt.Printf("got name: %s\n", data.name)

	// Attach native object to JS object
	// TODO: Type tagging
	handle := cgo.NewHandle(&data)
	// TODO: Don't leak finalizeData
	finalizeCb, finalizeData, _ := makeNapiFinalize(templateFinalize)
	status = C.napi_wrap(env.inner, thisArg, unsafe.Pointer(&handle), finalizeCb, finalizeData, nil)
	if status != C.napi_ok {
		panic(status)
	}

	return nil
}

func templateFinalize(env napiEnv, data unsafe.Pointer) {
	fmt.Printf("In Template finalize\n")
	handle := *(*cgo.Handle)(data)
	handle.Delete()
}

func templateMethodFoo(env napiEnv, info C.napi_callback_info) C.napi_value {
	this := templateMethodEntry(env, info)
	fmt.Println("name:", this.name)
	this.name = this.name + "-foo"
	fmt.Println("name:", this.name)
	return nil
}

func templateMethodBar(env napiEnv, info C.napi_callback_info) C.napi_value {
	return nil
}
