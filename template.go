package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"text/template"
	"unsafe"

	"github.com/drakedevel/go-text-template-napi/internal/napi"
)

var templateWrapper = napi.NewSafeWrapper[template.Template](0x1b339336b7154e7d, 0xa8cd781754bef7c9)

func extractString(env napi.Env, value napi.Value) (string, error) {
	// Get string length
	strLen, err := env.GetValueString(value, nil)
	if err != nil {
		return "", err
	}

	// Allocate buffer and get string contents
	buf := make([]byte, strLen+1)
	strLen, err = env.GetValueString(value, buf)
	if err != nil {
		return "", err
	}
	return string(buf[0:strLen]), nil
}

func extractBigint(env napi.Env, value napi.Value) (*big.Int, error) {
	// Get length
	wordCount := 0
	if err := env.GetValueBigintWords(value, nil, &wordCount, nil); err != nil {
		return nil, err
	}

	// Allocate space and get contents
	var signBit int
	words := make([]uint64, wordCount)
	if err := env.GetValueBigintWords(value, &signBit, &wordCount, &words[0]); err != nil {
		return nil, err
	}

	// Convert to big-endian bytes
	var buf bytes.Buffer
	for i := len(words) - 1; i >= 0; i-- {
		err := binary.Write(&buf, binary.BigEndian, words[i])
		if err != nil {
			return nil, err
		}
	}

	result := new(big.Int)
	result.SetBytes(buf.Bytes())
	if signBit > 0 {
		result.Neg(result)
	}
	return result, nil
}

func callbackEntry(env napi.Env, info napi.CallbackInfo, nArgs int) (napi.Value, int, []napi.Value, error) {
	var thisArg napi.Value
	argc := nArgs
	argv := make([]napi.Value, argc)
	var argvPtr *napi.Value
	if argc > 0 {
		argvPtr = &argv[0]
	}
	if err := env.GetCbInfo(info, &argc, argvPtr, &thisArg, nil); err != nil {
		return nil, 0, nil, err
	}
	return thisArg, argc, argv, nil
}

func templateMethodEntry(env napi.Env, info napi.CallbackInfo, nArgs int) (*template.Template, []napi.Value, error) {
	thisArg, _, argv, err := callbackEntry(env, info, nArgs)
	if err != nil {
		return nil, nil, err
	}

	// Retrieve wrapped native object from JS object
	this, err := templateWrapper.Unwrap(env, thisArg)
	if err != nil {
		return nil, nil, fmt.Errorf("object not correctly initialized: %w", err)
	}
	return this, argv, nil
}

type templateMethodFunc func(napi.Env, *template.Template, []napi.Value) (napi.Value, error)

func makeTemplateMethodCallback(fn templateMethodFunc, nArgs int) (napi.Callback, unsafe.Pointer, func()) {
	return napi.MakeNapiCallback(func(env napi.Env, info napi.CallbackInfo) (napi.Value, error) {
		this, args, err := templateMethodEntry(env, info, nArgs)
		if err != nil {
			return nil, err
		}
		return fn(env, this, args)
	})
}

func buildTemplateClass(env napi.Env) (napi.Value, error) {
	// Build property descriptors
	type method struct {
		fn    templateMethodFunc
		nArgs int
	}
	methods := map[string]method{
		// AddParseTree and ParseFS are unsupported
		"clone":            {templateMethodClone, 0},
		"definedTemplates": {templateMethodDefinedTemplates, 0},
		"delims":           {templateMethodDelims, 2},
		"execute":          {templateMethodExecute, 1},
		"executeTemplate":  {templateMethodExecuteTemplate, 2},
		"funcs":            {templateMethodFuncs, 1},
		"lookup":           {templateMethodLookup, 1},
		"name":             {templateMethodName, 0},
		"new":              {templateMethodNew, 1},
		"option":           {templateMethodOption, 1},
		"parse":            {templateMethodParse, 1},
		// TODO: ParseFiles
		// TODO: ParseGlob
		"templates": {templateMethodTemplates, 0},
	}
	var propDescs []napi.PropertyDescriptor
	for name, spec := range methods {
		// TODO: Don't leak cbData
		cb, cbData, _ := makeTemplateMethodCallback(spec.fn, spec.nArgs)
		nameObj, err := env.CreateString(name)
		if err != nil {
			return nil, err
		}
		propDescs = append(propDescs, napi.PropertyDescriptor{
			Name:       nameObj,
			Method:     cb,
			Attributes: napi.DefaultMethod,
			Data:       cbData,
		})
	}

	// Define class
	// TODO: Don't leak consData
	consCb, consData, _ := napi.MakeNapiCallback(templateConstructor)
	return env.DefineClass("Template", consCb, consData, propDescs)
}

func templateConstructor(env napi.Env, info napi.CallbackInfo) (napi.Value, error) {
	// TODO: Add check for new.target
	thisArg, argc, argv, err := callbackEntry(env, info, 1)
	if err != nil {
		return nil, err
	}

	// If no name was passed in, skip wrapping this object, and assume we'll
	// wrap it later with wrapExistingTemplate (e.g. in Clone).
	// TODO: Find way to give JS an error while allowing Clone/etc.
	if argc == 0 {
		return nil, err
	}

	name, err := extractString(env, argv[0])
	if err != nil {
		return nil, err
	}

	// Create native object and attach to JS object
	data := template.New(name)
	if err := templateWrapper.Wrap(env, thisArg, data); err != nil {
		return nil, err
	}
	return nil, nil
}

func wrapExistingTemplate(env napi.Env, tmpl *template.Template) (napi.Value, error) {
	instData, err := getInstanceData(env)
	if err != nil {
		return nil, err
	}
	constructor, err := env.GetReferenceValue(instData.templateConstructor)
	if err != nil {
		return nil, err
	}
	instance, err := env.NewInstance(constructor, nil)
	if err != nil {
		return nil, err
	}
	if err := templateWrapper.Wrap(env, instance, tmpl); err != nil {
		return nil, err
	}
	return instance, nil
}

func templateMethodClone(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	cloned, err := this.Clone()
	if err != nil {
		return nil, err
	}
	return wrapExistingTemplate(env, cloned)
}

func templateMethodDefinedTemplates(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	return env.CreateString(this.DefinedTemplates())
}

func templateMethodDelims(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	left, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	right, err := extractString(env, args[1])
	if err != nil {
		return nil, err
	}
	this.Delims(left, right)
	return nil, nil // XXX: Should return this
}

func templateMethodExecute(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	// TODO: Allow passing in a stream?
	data, err := jsValueToGo(env, args[0])
	if err != nil {
		return nil, err
	}
	modData, err := getInstanceData(env)
	if err != nil {
		return nil, err
	}
	modData.envStack.Enter(env)
	defer modData.envStack.Exit(env)
	var buf bytes.Buffer
	if err := this.Execute(&buf, data); err != nil {
		// TODO: Map to better JS error?
		return nil, err
	}
	return env.CreateString(buf.String())
}

func templateMethodExecuteTemplate(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	// TODO: Allow passing in a stream?
	name, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	data, err := jsValueToGo(env, args[1])
	if err != nil {
		return nil, err
	}
	modData, err := getInstanceData(env)
	if err != nil {
		return nil, err
	}
	modData.envStack.Enter(env)
	defer modData.envStack.Exit(env)
	var buf bytes.Buffer
	if err := this.ExecuteTemplate(&buf, name, data); err != nil {
		// TODO: Map to better JS error?
		return nil, err
	}
	return env.CreateString(buf.String())
}

func makeJsCallback(envStack *envStack, jsFnRef napi.Ref) interface{} {
	return func(args ...interface{}) (interface{}, error) {
		env := envStack.Current()
		jsFn, err := env.GetReferenceValue(jsFnRef)
		if err != nil {
			return nil, err
		}
		undefVal, err := env.GetUndefined()
		if err != nil {
			return nil, err
		}
		jsArgs := make([]napi.Value, len(args))
		for i, arg := range args {
			jsArg, err := goValueToJs(env, arg)
			if err != nil {
				return nil, err
			}
			jsArgs[i] = jsArg
		}
		result, err := env.CallFunction(undefVal, jsFn, jsArgs)
		if err != nil {
			return nil, err
		}
		return jsValueToGo(env, result)
	}
}

func templateMethodFuncs(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	modData, err := getInstanceData(env)
	if err != nil {
		return nil, err
	}
	propNames, err := env.GetPropertyNames(args[0])
	if err != nil {
		return nil, err
	}
	length, err := env.GetArrayLength(propNames)
	if err != nil {
		return nil, err
	}
	funcMap := make(template.FuncMap)
	for i := uint32(0); i < length; i++ {
		// TODO: Scope?
		propNameValue, err := env.GetElement(propNames, i)
		if err != nil {
			return nil, err
		}
		propName, err := extractString(env, propNameValue)
		if err != nil {
			return nil, err
		}
		propValue, err := env.GetProperty(args[0], propNameValue)
		if err != nil {
			return nil, err
		}
		propType, err := env.Typeof(propValue)
		if err != nil {
			return nil, err
		}
		if propType == napi.Undefined {
			continue
		}
		if propType != napi.Function {
			// TODO: Custom error mechanism
			excMsg := fmt.Sprintf("Key '%s' is not a function", propName)
			err = env.ThrowTypeError("ERR_INVALID_ARG_TYPE", excMsg)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("threw exception")
		}
		// TODO: Don't leak propRef
		propRef, err := env.CreateReference(propValue, 1)
		if err != nil {
			return nil, err
		}
		funcMap[propName] = makeJsCallback(&modData.envStack, propRef)
	}
	this.Funcs(funcMap)
	return nil, nil // XXX: Should return this
}

func templateMethodLookup(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	name, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	result := this.Lookup(name)
	if result == nil {
		return nil, nil
	}
	return wrapExistingTemplate(env, result)
}

func templateMethodName(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	return env.CreateString(this.Name())
}

func templateMethodNew(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	name, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	return wrapExistingTemplate(env, this.New(name))
}

func templateMethodOption(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	// XXX: Should be variadic
	option, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	this.Option(option)
	return nil, nil // XXX: Should return this
}

func templateMethodParse(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	text, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	result, err := this.Parse(text)
	if err != nil {
		// TODO: Map to better JS error?
		return nil, err
	}
	if result != this {
		panic("Expected Parse to return itself")
	}
	return nil, nil // XXX: Should return this
}

func templateMethodTemplates(env napi.Env, this *template.Template, args []napi.Value) (napi.Value, error) {
	templates := this.Templates()
	result, err := env.CreateArrayWithLength(len(templates))
	if err != nil {
		return nil, err
	}
	for i, template := range templates {
		wrapped, err := wrapExistingTemplate(env, template)
		if err != nil {
			return nil, err
		}
		if err := env.SetElement(result, uint32(i), wrapped); err != nil {
			return nil, err
		}
	}
	return result, nil
}
