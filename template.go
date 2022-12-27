package main

import (
	"fmt"
	"runtime/cgo"
	"unsafe"

	"github.com/drakedevel/go-text-template-napi/internal/napi"
)

type template struct {
	name string
}

func extractString(env napi.Env, value napi.Value) string {
	// Get string length
	strLen := env.GetValueString(value, nil)

	// Allocate buffer and get string contents
	buf := make([]byte, strLen+1)
	strLen = env.GetValueString(value, buf)
	return string(buf[0:strLen])
}

func templateMethodEntry(env napi.Env, info napi.CallbackInfo) (this *template) {
	// TODO: Process args?
	var thisArg napi.Value
	env.GetCbInfo(info, nil, nil, &thisArg, nil)

	// Retrieve wrapped native object from JS object
	// TODO: Type tagging
	wrapped := env.Unwrap(thisArg)
	handle := *(*cgo.Handle)(wrapped)

	return handle.Value().(*template)
}

func buildTemplateClass(env napi.Env) napi.Value {
	// Build property descriptors
	var propDescs []napi.PropertyDescriptor

	// TODO: Don't leak fooData
	fooCb, fooData, _ := napi.MakeNapiCallback(templateMethodFoo)
	propDescs = append(propDescs, napi.PropertyDescriptor{
		Name:       env.CreateString("foo"),
		Method:     fooCb,
		Attributes: napi.DefaultMethod,
		Data:       fooData,
	})

	// TODO: Don't leak barData
	barCb, barData, _ := napi.MakeNapiCallback(templateMethodBar)
	propDescs = append(propDescs, napi.PropertyDescriptor{
		Name:       env.CreateString("bar"),
		Method:     barCb,
		Attributes: napi.DefaultMethod,
		Data:       barData,
	})

	// Define class
	// TODO: Don't leak consData
	consCb, consData, _ := napi.MakeNapiCallback(templateConstructor)
	return env.DefineClass("Template", consCb, consData, propDescs)
}

func templateConstructor(env napi.Env, info napi.CallbackInfo) napi.Value {
	argc := 1
	argv := make([]napi.Value, 1)
	var thisArg napi.Value
	env.GetCbInfo(info, &argc, &argv[0], &thisArg, nil)
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
	finalizeCb, finalizeData, _ := napi.MakeNapiFinalize(templateFinalize)
	env.Wrap(thisArg, unsafe.Pointer(&handle), finalizeCb, finalizeData)

	return nil
}

func templateFinalize(env napi.Env, data unsafe.Pointer) {
	fmt.Printf("In Template finalize\n")
	handle := *(*cgo.Handle)(data)
	handle.Delete()
}

func templateMethodFoo(env napi.Env, info napi.CallbackInfo) napi.Value {
	this := templateMethodEntry(env, info)
	fmt.Println("name:", this.name)
	this.name = this.name + "-foo"
	fmt.Println("name:", this.name)
	return nil
}

func templateMethodBar(env napi.Env, info napi.CallbackInfo) napi.Value {
	return nil
}
