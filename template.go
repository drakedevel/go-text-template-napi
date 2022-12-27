package main

import (
	"bytes"
	"fmt"
	"runtime/cgo"
	"text/template"
	"unsafe"

	"github.com/drakedevel/go-text-template-napi/internal/napi"
)

type templateObj struct {
	inner *template.Template
}

func (tmpl *templateObj) wrap(env napi.Env, object napi.Value) {
	// TODO: Type tagging
	handle := cgo.NewHandle(tmpl)
	// TODO: Don't leak finalizeData
	finalizeCb, finalizeData, _ := napi.MakeNapiFinalize(templateFinalize)
	// FIXME: Don't use Handle pointer
	env.Wrap(object, unsafe.Pointer(&handle), finalizeCb, finalizeData)
}

func unwrapTemplate(env napi.Env, object napi.Value) *templateObj {
	// TODO: Type tagging
	wrapped := env.Unwrap(object)
	// TODO: Don't use Handle pointer
	handle := *(*cgo.Handle)(wrapped)
	return handle.Value().(*templateObj)
}

func extractString(env napi.Env, value napi.Value) string {
	// Get string length
	strLen := env.GetValueString(value, nil)

	// Allocate buffer and get string contents
	buf := make([]byte, strLen+1)
	strLen = env.GetValueString(value, buf)
	return string(buf[0:strLen])
}

func templateMethodEntry(env napi.Env, info napi.CallbackInfo, nArgs int) (this *templateObj, args []napi.Value) {
	var thisArg napi.Value
	argc := nArgs
	argv := make([]napi.Value, argc)
	var argvPtr *napi.Value
	if argc > 0 {
		argvPtr = &argv[0]
	}
	env.GetCbInfo(info, &argc, argvPtr, &thisArg, nil)
	if argc != nArgs {
		// TODO: Throw
		panic("wrong argument count")
	}

	// Retrieve wrapped native object from JS object
	return unwrapTemplate(env, thisArg), argv
}

type templateMethodFunc func(napi.Env, *templateObj, []napi.Value) napi.Value

func makeTemplateMethodCallback(fn templateMethodFunc, nArgs int) (napi.Callback, unsafe.Pointer, func()) {
	return napi.MakeNapiCallback(func(env napi.Env, info napi.CallbackInfo) napi.Value {
		this, args := templateMethodEntry(env, info, nArgs)
		return fn(env, this, args)
	})
}

func buildTemplateClass(env napi.Env) napi.Value {
	// Build property descriptors
	var propDescs []napi.PropertyDescriptor

	addMethod := func(name string, fn templateMethodFunc, nArgs int) {
		// TODO: Don't leak cbData
		cb, cbData, _ := makeTemplateMethodCallback(fn, nArgs)
		propDescs = append(propDescs, napi.PropertyDescriptor{
			Name:       env.CreateString(name),
			Method:     cb,
			Attributes: napi.DefaultMethod,
			Data:       cbData,
		})
	}

	// TODO: AddParseTree?
	// TODO: Clone
	// TODO: DefinedTemplates?
	// TODO: Delims
	addMethod("execute", templateMethodExecute, 1)
	addMethod("executeTemplate", templateMethodExecuteTemplate, 2)
	// TODO: Funcs
	// TODO: Lookup
	addMethod("name", templateMethodName, 0)
	// TODO: New
	// TODO: Option
	addMethod("parse", templateMethodParse, 1)
	// TODO: ParseFS
	// TODO: ParseFiles
	// TODO: ParseGlob
	// TODO: Templates

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

	// Create native object and attach to JS object
	data := templateObj{template.New(extractString(env, argv[0]))}
	data.wrap(env, thisArg)

	return nil
}

func templateFinalize(env napi.Env, data unsafe.Pointer) {
	fmt.Printf("In Template finalize\n")
	handle := *(*cgo.Handle)(data)
	handle.Delete()
}

func convertTemplateData(env napi.Env, value napi.Value) interface{} {
	valueType := env.Typeof(value)
	switch valueType {
	case napi.Undefined, napi.Null:
		// TODO: Filter out Undefined from parent object, keep Null
		return nil
	// TODO: case napi.Boolean:
	// TODO: case napi.Number:
	case napi.String:
		return extractString(env, value)
	// TODO: case napi.Symbol:
	case napi.Object:
		if env.IsArray(value) {
			length := env.GetArrayLength(value)
			result := make([]interface{}, length)
			var i uint32
			for i = 0; i < length; i++ {
				// TODO: Scope?
				result[i] = convertTemplateData(env, env.GetElement(value, i))
			}
			return result
		} else {
			// TODO: Should any other object types get special handling?
			// TODO: Include prototypes with function support
			propNames := env.GetAllPropertyNames(value, napi.KeyOwnOnly, napi.KeySkipSymbols, napi.KeyNumbersToStrings)
			length := env.GetArrayLength(propNames)
			result := map[string]interface{}{}
			var i uint32
			for i = 0; i < length; i++ {
				// TODO: Scope?
				key := env.GetElement(propNames, i)
				result[extractString(env, key)] = convertTemplateData(env, env.GetProperty(value, key))
			}
			return result
		}
	// TODO: case napi.Function:
	// TODO: case napi.External:
	// TODO: case napi.Bigint:
	default:
		fmt.Printf("unsupported type %v\n", valueType)
		panic("unknown value type")
	}
}

func templateMethodExecute(env napi.Env, this *templateObj, args []napi.Value) napi.Value {
	// TODO: Allow passing in a stream?
	var buf bytes.Buffer
	err := this.inner.Execute(&buf, convertTemplateData(env, args[0]))
	if err != nil {
		// TODO: Throw
		panic(err)
	}
	return env.CreateString(buf.String())
}

func templateMethodExecuteTemplate(env napi.Env, this *templateObj, args []napi.Value) napi.Value {
	// TODO: Allow passing in a stream?
	var buf bytes.Buffer
	err := this.inner.ExecuteTemplate(&buf, extractString(env, args[0]), convertTemplateData(env, args[1]))
	if err != nil {
		// TODO: Throw
		panic(err)
	}
	return env.CreateString(buf.String())
}

func templateMethodName(env napi.Env, this *templateObj, args []napi.Value) napi.Value {
	return env.CreateString(this.inner.Name())
}

func templateMethodParse(env napi.Env, this *templateObj, args []napi.Value) napi.Value {
	result, err := this.inner.Parse(extractString(env, args[0]))
	if err != nil {
		// TODO: Throw
		panic(err)
	}
	if result != this.inner {
		panic("Expected Parse to return itself")
	}
	return nil // XXX: Should return this
}
