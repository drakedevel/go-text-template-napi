package main

import (
	"bytes"
	"fmt"
	"text/template"
	"unsafe"

	"github.com/Masterminds/sprig/v3"
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

func (ta *templateAssn) RemoveFunctionRef(name string) napi.Ref {
	// While there's no way to delete a Func once one has been added, we may
	// need to delete a ref if we replace a JS function with a native one.
	if oldRef, ok := ta.funcRefs[name]; ok {
		delete(ta.funcRefs, name)
		return oldRef
	}
	return nil
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

func callbackEntry(env napi.Env, info napi.CallbackInfo, minArgs int) (napi.Value, []napi.Value, error) {
	// Get argument count
	argc := 0
	if err := env.GetCbInfo(info, &argc, nil, nil, nil); err != nil {
		return nil, nil, err
	}

	// If missing required arguments, pad the slice so we get undefined values
	if argc < minArgs {
		argc = minArgs
	}
	argv := make([]napi.Value, argc)

	// Fetch thisArg and all arguments
	var thisArg napi.Value
	var argvPtr *napi.Value
	if len(argv) > 0 {
		argvPtr = &argv[0]
	}
	if err := env.GetCbInfo(info, &argc, argvPtr, &thisArg, nil); err != nil {
		return nil, nil, err
	}
	return thisArg, argv, nil
}

type templateMethodFunc func(*jsTemplate, napi.Env, []napi.Value) (napi.Value, error)

func makeTemplateMethodCallback(fn templateMethodFunc, minArgs int, chain bool) (napi.Callback, unsafe.Pointer, func()) {
	return napi.MakeNapiCallback(func(env napi.Env, info napi.CallbackInfo) (napi.Value, error) {
		thisArg, args, err := callbackEntry(env, info, minArgs)
		if err != nil {
			return nil, err
		}

		// Retrieve wrapped native object from JS object
		this, err := templateWrapper.Unwrap(env, thisArg)
		if err != nil {
			return nil, fmt.Errorf("object not correctly initialized: %w", err)
		}

		result, err := fn(this, env, args)
		if err != nil {
			return nil, err
		}
		if chain {
			result = thisArg
		}
		return result, nil
	})
}

type staticMethodFunc func(napi.Env, []napi.Value) (napi.Value, error)

func makeStaticMethodCallback(fn staticMethodFunc, minArgs int) (napi.Callback, unsafe.Pointer, func()) {
	return napi.MakeNapiCallback(func(env napi.Env, info napi.CallbackInfo) (napi.Value, error) {
		_, args, err := callbackEntry(env, info, minArgs)
		if err != nil {
			return nil, err
		}
		return fn(env, args)
	})
}

func buildTemplateClass(env napi.Env, clsName string) (napi.Value, error) {
	// Build property descriptors
	type method struct {
		fn      templateMethodFunc
		minArgs int
		chain   bool
	}
	type staticMethod struct {
		fn      staticMethodFunc
		minArgs int
	}
	methods := map[string]method{
		// AddParseTree and ParseFS are unsupported
		// Execute and ExecuteTemplates are supported with string returns
		"clone":                 {(*jsTemplate).methodClone, 0, false},
		"definedTemplates":      {(*jsTemplate).methodDefinedTemplates, 0, false},
		"delims":                {(*jsTemplate).methodDelims, 2, true},
		"executeString":         {(*jsTemplate).methodExecuteString, 1, false},
		"executeTemplateString": {(*jsTemplate).methodExecuteTemplateString, 2, false},
		"funcs":                 {(*jsTemplate).methodFuncs, 1, true},
		"lookup":                {(*jsTemplate).methodLookup, 1, false},
		"name":                  {(*jsTemplate).methodName, 0, false},
		"new":                   {(*jsTemplate).methodNew, 1, false},
		"option":                {(*jsTemplate).methodOption, 0, true},
		"parse":                 {(*jsTemplate).methodParse, 1, true},
		"parseFiles":            {(*jsTemplate).methodParseFiles, 0, true},
		"parseGlob":             {(*jsTemplate).methodParseGlob, 1, true},
		"templates":             {(*jsTemplate).methodTemplates, 0, false},

		// These functions are not part of the text/template API
		"addSprigFuncs":         {(*jsTemplate).methodAddSprigFuncs, 0, true},
		"addSprigHermeticFuncs": {(*jsTemplate).methodAddSprigHermeticFuncs, 0, true},
	}
	staticMethods := map[string]staticMethod{
		// ParseFS is unsupported
		"parseFiles": {staticTemplateParseFiles, 0},
		"parseGlob":  {staticTemplateParseGlob, 1},
	}
	var propDescs []napi.PropertyDescriptor
	for name, spec := range methods {
		// TODO: Don't leak cbData
		cb, cbData, _ := makeTemplateMethodCallback(spec.fn, spec.minArgs, spec.chain)
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
	for name, spec := range staticMethods {
		// TODO: Don't leak cbData
		cb, cbData, _ := makeStaticMethodCallback(spec.fn, spec.minArgs)
		nameObj, err := env.CreateString(name)
		if err != nil {
			return nil, err
		}
		propDescs = append(propDescs, napi.PropertyDescriptor{
			Name:       nameObj,
			Method:     cb,
			Attributes: napi.Static | napi.DefaultMethod,
			Data:       cbData,
		})
	}

	// Define class
	// TODO: Don't leak consData
	consCb, consData, _ := napi.MakeNapiCallback(templateConstructor)
	return env.DefineClass(clsName, consCb, consData, propDescs)
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
	thisArg, argv, err := callbackEntry(env, info, 0)
	if err != nil {
		return nil, err
	}

	// If no name was passed in, skip wrapping this object, and assume we'll
	// wrap it later with wrapExistingTemplate (e.g. in Clone).
	// TODO: Find way to give JS an error while allowing Clone/etc.
	if len(argv) == 0 {
		return nil, err
	}

	name, err := jsStringToGo(env, argv[0])
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
	left, err := jsStringToGo(env, args[0])
	if err != nil {
		return nil, err
	}
	right, err := jsStringToGo(env, args[1])
	if err != nil {
		return nil, err
	}
	jst.inner.Delims(left, right)
	return nil, nil
}

func (jst *jsTemplate) methodExecuteString(env napi.Env, args []napi.Value) (napi.Value, error) {
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

func (jst *jsTemplate) methodExecuteTemplateString(env napi.Env, args []napi.Value) (napi.Value, error) {
	// TODO: Allow passing in a stream?
	name, err := jsStringToGo(env, args[0])
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

func panicToErr(err *error) {
	if r := recover(); r != nil {
		*err = fmt.Errorf("caught panic: %v", r)
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
	// Create references and closures for all passed-in functions
	// TODO: Leaks if errors occcur before Funcs succeeds
	refMap := make(map[string]napi.Ref)
	funcMap := make(template.FuncMap)
	for i := uint32(0); i < length; i++ {
		// TODO: Scope?
		propNameValue, err := env.GetElement(propNames, i)
		if err != nil {
			return nil, err
		}
		propName, err := jsStringToGo(env, propNameValue)
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
		funcMap[propName] = makeJsCallback(&modData.envStack, propRef)
	}

	// Funcs panics if the caller passes in an invalid name, so catch that
	// and convert it to a normal error.
	err = func() (err error) {
		defer panicToErr(&err)
		jst.inner.Funcs(funcMap)
		return
	}()
	if err != nil {
		return nil, err
	}

	// Save new references, and unreference any replaced functions
	for name, ref := range refMap {
		oldRef := jst.assn.AddFunctionRef(name, ref)
		if oldRef != nil {
			// Swallow errors here since we can't do anything about them
			_, _ = env.ReferenceUnref(oldRef)
		}
	}

	return nil, nil
}

func (jst *jsTemplate) methodLookup(env napi.Env, args []napi.Value) (napi.Value, error) {
	name, err := jsStringToGo(env, args[0])
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
	name, err := jsStringToGo(env, args[0])
	if err != nil {
		return nil, err
	}
	return wrapExistingTemplate(env, jst.inner.New(name), jst.assn)
}

func (jst *jsTemplate) methodOption(env napi.Env, args []napi.Value) (napi.Value, error) {
	options, err := jsValuesToGo(env, args, jsStringToGo)
	if err != nil {
		return nil, err
	}
	// Option panics if the string is invalid, return an error instead
	err = func() (err error) {
		defer panicToErr(&err)
		jst.inner.Option(options...)
		return
	}()
	return nil, err
}

func (jst *jsTemplate) methodParse(env napi.Env, args []napi.Value) (napi.Value, error) {
	text, err := jsStringToGo(env, args[0])
	if err != nil {
		return nil, err
	}

	if _, err := jst.inner.Parse(text); err != nil {
		// TODO: Map to better JS error?
		return nil, err
	}
	return nil, nil
}

func (jst *jsTemplate) methodParseFiles(env napi.Env, args []napi.Value) (napi.Value, error) {
	files, err := jsValuesToGo(env, args, jsStringToGo)
	if err != nil {
		return nil, err
	}
	if _, err := jst.inner.ParseFiles(files...); err != nil {
		// TODO: Map to better JS error?
		return nil, err
	}
	return nil, nil
}

func (jst *jsTemplate) methodParseGlob(env napi.Env, args []napi.Value) (napi.Value, error) {
	text, err := jsStringToGo(env, args[0])
	if err != nil {
		return nil, err
	}
	if _, err := jst.inner.ParseGlob(text); err != nil {
		// TODO: Map to better JS error?
		return nil, err
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

func (jst *jsTemplate) addNativeFuncs(env napi.Env, funcs template.FuncMap) error {
	// Add the native functions
	jst.inner.Funcs(funcs)

	// Unreference any JS functions these replaced
	for name := range funcs {
		oldRef := jst.assn.RemoveFunctionRef(name)
		if oldRef != nil {
			// Swallow errors here since we can't do anything about them
			_, _ = env.ReferenceUnref(oldRef)
		}
	}
	return nil
}

func (jst *jsTemplate) methodAddSprigFuncs(env napi.Env, args []napi.Value) (napi.Value, error) {
	err := jst.addNativeFuncs(env, sprig.TxtFuncMap())
	return nil, err
}

func (jst *jsTemplate) methodAddSprigHermeticFuncs(env napi.Env, args []napi.Value) (napi.Value, error) {
	err := jst.addNativeFuncs(env, sprig.HermeticTxtFuncMap())
	return nil, err
}

func staticTemplateParseFiles(env napi.Env, args []napi.Value) (napi.Value, error) {
	files, err := jsValuesToGo(env, args, jsStringToGo)
	if err != nil {
		return nil, err
	}
	result, err := template.ParseFiles(files...)
	if err != nil {
		return nil, err
	}
	return wrapExistingTemplate(env, result, newTemplateAssn())
}

func staticTemplateParseGlob(env napi.Env, args []napi.Value) (napi.Value, error) {
	glob, err := jsStringToGo(env, args[0])
	if err != nil {
		return nil, err
	}
	result, err := template.ParseGlob(glob)
	if err != nil {
		return nil, err
	}
	return wrapExistingTemplate(env, result, newTemplateAssn())
}
