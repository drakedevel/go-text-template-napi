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

// templateAssn represents the "association" of Template objects. All jsTemplates
// wrapping associated Templates must have the same templateAssn pointer.
type templateAssn struct {
	// funcRefs holds the JS references for all JS functions passed to the
	// Funcs method. The JS-internal refcount is used to track how many
	// templateAssns (not jsTemplates) reference each function.
	funcRefs map[string]napi.Ref

	// refCount tracks the number of jsTemplates referring to this object.
	// This has to be done manually so we can Unref references in funcRefs
	// as soon as it hits zero in a jsTemplate finalize call, while we still
	// have a napi.Env available.
	refCount uint
}

func newTemplateAssn() *templateAssn {
	return &templateAssn{make(map[string]napi.Ref), 0}
}

func (ta *templateAssn) AddFunctionRef(name string, ref napi.Ref) napi.Ref {
	var result napi.Ref
	if oldRef, ok := ta.funcRefs[name]; ok {
		result = oldRef
	}
	ta.funcRefs[name] = ref
	return result
}

func (ta *templateAssn) Clone(env napi.Env) (*templateAssn, error) {
	// TODO: Leaks references if there's an error part-way through
	result := newTemplateAssn()
	for name, ref := range ta.funcRefs {
		result.AddFunctionRef(name, ref)
		if _, err := env.ReferenceRef(ref); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func (ta *templateAssn) MaybeFinalize(env napi.Env) error {
	if ta.refCount > 0 {
		return nil
	}
	for _, ref := range ta.funcRefs {
		if _, err := env.ReferenceUnref(ref); err != nil {
			return err
		}
	}
	return nil
}

func (ta *templateAssn) Ref(jst *jsTemplate) {
	if jst.assn != nil {
		panic("Tried to associate template with multiple associations")
	}
	ta.refCount++
	jst.assn = ta
}

func (ta *templateAssn) Unref(jst *jsTemplate) {
	if ta.refCount == 0 {
		panic("Tried to unreference template association with 0 references")
	}
	if jst.assn != ta {
		panic("Tried to unreference template from wrong association")
	}
	jst.assn = nil
	ta.refCount--
}

type jsTemplate struct {
	inner *template.Template
	assn  *templateAssn
}

var templateWrapper = napi.NewSafeWrapper[jsTemplate](0x1b339336b7154e7d, 0xa8cd781754bef7c9)

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

type templateMethodFunc func(*jsTemplate, napi.Env, []napi.Value) (napi.Value, error)

func makeTemplateMethodCallback(fn templateMethodFunc, nArgs int, chain bool) (napi.Callback, unsafe.Pointer, func()) {
	return napi.MakeNapiCallback(func(env napi.Env, info napi.CallbackInfo) (napi.Value, error) {
		thisArg, _, argv, err := callbackEntry(env, info, nArgs)
		if err != nil {
			return nil, err
		}

		// Retrieve wrapped native object from JS object
		this, err := templateWrapper.Unwrap(env, thisArg)
		if err != nil {
			return nil, fmt.Errorf("object not correctly initialized: %w", err)
		}

		result, err := fn(this, env, argv)
		if err != nil {
			return nil, err
		}
		if chain {
			result = thisArg
		}
		return result, nil
	})
}

func buildTemplateClass(env napi.Env) (napi.Value, error) {
	// Build property descriptors
	type method struct {
		fn    templateMethodFunc
		nArgs int
		chain bool
	}
	methods := map[string]method{
		// AddParseTree and ParseFS are unsupported
		"clone":            {(*jsTemplate).methodClone, 0, false},
		"definedTemplates": {(*jsTemplate).methodDefinedTemplates, 0, false},
		"delims":           {(*jsTemplate).methodDelims, 2, true},
		"execute":          {(*jsTemplate).methodExecute, 1, false},
		"executeTemplate":  {(*jsTemplate).methodExecuteTemplate, 2, false},
		"funcs":            {(*jsTemplate).methodFuncs, 1, true},
		"lookup":           {(*jsTemplate).methodLookup, 1, false},
		"name":             {(*jsTemplate).methodName, 0, false},
		"new":              {(*jsTemplate).methodNew, 1, false},
		"option":           {(*jsTemplate).methodOption, 1, true},
		"parse":            {(*jsTemplate).methodParse, 1, true},
		// TODO: ParseFiles
		// TODO: ParseGlob
		"templates": {(*jsTemplate).methodTemplates, 0, false},
	}
	var propDescs []napi.PropertyDescriptor
	for name, spec := range methods {
		// TODO: Don't leak cbData
		cb, cbData, _ := makeTemplateMethodCallback(spec.fn, spec.nArgs, spec.chain)
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

func wrapTemplateObject(env napi.Env, object napi.Value, tmpl *template.Template, assn *templateAssn) error {
	jst := &jsTemplate{tmpl, nil}
	if err := templateWrapper.Wrap(env, object, jst, templateFinalize); err != nil {
		return err
	}
	// Wait until after wrapping succeeds to reference the association to
	// ensure the finalizer will be called.
	assn.Ref(jst)
	return nil
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
	if err := wrapTemplateObject(env, thisArg, template.New(name), newTemplateAssn()); err != nil {
		return nil, err
	}
	return nil, nil
}

func templateFinalize(env napi.Env, data interface{}) error {
	jst := data.(*jsTemplate)
	assn := jst.assn
	if assn == nil {
		return nil
	}
	assn.Unref(jst)
	return assn.MaybeFinalize(env)
}

func wrapExistingTemplate(env napi.Env, tmpl *template.Template, assn *templateAssn) (napi.Value, error) {
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
	if err := wrapTemplateObject(env, instance, tmpl, assn); err != nil {
		return nil, err
	}
	return instance, nil
}

func (jst *jsTemplate) methodClone(env napi.Env, args []napi.Value) (napi.Value, error) {
	clonedTmpl, err := jst.inner.Clone()
	if err != nil {
		return nil, err
	}
	clonedAssn, err := jst.assn.Clone(env)
	if err != nil {
		return nil, err
	}
	return wrapExistingTemplate(env, clonedTmpl, clonedAssn)
}

func (jst *jsTemplate) methodDefinedTemplates(env napi.Env, args []napi.Value) (napi.Value, error) {
	return env.CreateString(jst.inner.DefinedTemplates())
}

func (jst *jsTemplate) methodDelims(env napi.Env, args []napi.Value) (napi.Value, error) {
	left, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	right, err := extractString(env, args[1])
	if err != nil {
		return nil, err
	}
	jst.inner.Delims(left, right)
	return nil, nil
}

func (jst *jsTemplate) methodExecute(env napi.Env, args []napi.Value) (napi.Value, error) {
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
	if err := jst.inner.Execute(&buf, data); err != nil {
		// TODO: Map to better JS error?
		return nil, err
	}
	return env.CreateString(buf.String())
}

func (jst *jsTemplate) methodExecuteTemplate(env napi.Env, args []napi.Value) (napi.Value, error) {
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
	if err := jst.inner.ExecuteTemplate(&buf, name, data); err != nil {
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

func (jst *jsTemplate) methodFuncs(env napi.Env, args []napi.Value) (napi.Value, error) {
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
	// Create references for all passed-in functions
	// TODO: Leaks if errors occcur part-way through this loop
	refMap := make(map[string]napi.Ref)
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
		propRef, err := env.CreateReference(propValue, 1)
		if err != nil {
			return nil, err
		}
		refMap[propName] = propRef
	}

	// Create closures and pass them to Funcs
	funcMap := make(template.FuncMap)
	var oldRefs []napi.Ref
	for name, ref := range refMap {
		funcMap[name] = makeJsCallback(&modData.envStack, ref)
		oldRef := jst.assn.AddFunctionRef(name, ref)
		if oldRef != nil {
			oldRefs = append(oldRefs, oldRef)
		}
	}
	jst.inner.Funcs(funcMap)

	// Clean up old references
	// TODO: Leaks if an error occurs in the middle
	for _, ref := range oldRefs {
		if _, err := env.ReferenceUnref(ref); err != nil {
			return nil, err
		}
	}

	return nil, nil
}

func (jst *jsTemplate) methodLookup(env napi.Env, args []napi.Value) (napi.Value, error) {
	name, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	result := jst.inner.Lookup(name)
	if result == nil {
		return nil, nil
	}
	return wrapExistingTemplate(env, result, jst.assn)
}

func (jst *jsTemplate) methodName(env napi.Env, args []napi.Value) (napi.Value, error) {
	return env.CreateString(jst.inner.Name())
}

func (jst *jsTemplate) methodNew(env napi.Env, args []napi.Value) (napi.Value, error) {
	name, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	return wrapExistingTemplate(env, jst.inner.New(name), jst.assn)
}

func (jst *jsTemplate) methodOption(env napi.Env, args []napi.Value) (napi.Value, error) {
	// XXX: Should be variadic
	option, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	jst.inner.Option(option)
	return nil, nil
}

func (jst *jsTemplate) methodParse(env napi.Env, args []napi.Value) (napi.Value, error) {
	text, err := extractString(env, args[0])
	if err != nil {
		return nil, err
	}
	result, err := jst.inner.Parse(text)
	if err != nil {
		// TODO: Map to better JS error?
		return nil, err
	}
	if result != jst.inner {
		panic("Expected Parse to return itself")
	}
	return nil, nil
}

func (jst *jsTemplate) methodTemplates(env napi.Env, args []napi.Value) (napi.Value, error) {
	templates := jst.inner.Templates()
	result, err := env.CreateArrayWithLength(len(templates))
	if err != nil {
		return nil, err
	}
	for i, template := range templates {
		wrapped, err := wrapExistingTemplate(env, template, jst.assn)
		if err != nil {
			return nil, err
		}
		if err := env.SetElement(result, uint32(i), wrapped); err != nil {
			return nil, err
		}
	}
	return result, nil
}
