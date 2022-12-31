package main

import (
	"text/template"

	"github.com/drakedevel/go-text-template-napi/internal/napi"
)

type moduleData struct {
	templateConstructor napi.Ref
	envStack            envStack
}

func getInstanceData(env napi.Env) (*moduleData, error) {
	raw, err := napi.GetInstanceData(env)
	if err != nil {
		return nil, err
	}
	return raw.(*moduleData), nil
}

func moduleTeardown(env napi.Env, data interface{}) error {
	modData := data.(*moduleData)
	if err := env.DeleteReference(modData.templateConstructor); err != nil {
		return err
	}
	return nil
}

type propBuilder func(napi.Env, string) (napi.Value, error)

func makeHelperBuilder(impl staticMethodFunc, minArgs int) propBuilder {
	return func(env napi.Env, name string) (napi.Value, error) {
		// TODO: Don't leak cbData
		cb, cbData, _ := makeStaticMethodCallback(impl, minArgs)
		return env.CreateFunction(name, cbData, cb)
	}
}

func makeEscapeStringBuilder(fn func(string) string) propBuilder {
	return makeHelperBuilder(func(env napi.Env, args []napi.Value) (napi.Value, error) {
		str, err := extractString(env, args[0])
		if err != nil {
			return nil, err
		}
		return env.CreateString(fn(str))
	}, 1)
}

func makeEscaperBuilder(fn func(...any) string) propBuilder {
	return makeHelperBuilder(func(env napi.Env, args []napi.Value) (napi.Value, error) {
		goArgs, err := extractArray(env, args, jsValueToGo)
		if err != nil {
			return nil, err
		}
		return env.CreateString(fn(goArgs...))
	}, 0)
}

func moduleInit(env napi.Env, exports napi.Value) (napi.Value, error) {
	// Build module properties values
	propBuilders := map[string]propBuilder{
		"Template":         buildTemplateClass,
		"htmlEscapeString": makeEscapeStringBuilder(template.HTMLEscapeString),
		"htmlEscaper":      makeEscaperBuilder(template.HTMLEscaper),
		"jsEscapeString":   makeEscapeStringBuilder(template.JSEscapeString),
		"jsEscaper":        makeEscaperBuilder(template.JSEscaper),
		"urlQueryEscaper":  makeEscaperBuilder(template.URLQueryEscaper),
	}
	propValues := make(map[string]napi.Value)
	for name, builder := range propBuilders {
		propValue, err := builder(env, name)
		if err != nil {
			return nil, err
		}
		propValues[name] = propValue
	}

	// Build property descriptors and create properties
	var propDescs []napi.PropertyDescriptor
	for name, propValue := range propValues {
		nameValue, err := env.CreateString(name)
		if err != nil {
			return nil, err
		}
		propDescs = append(propDescs, napi.PropertyDescriptor{
			Name:       nameValue,
			Value:      propValue,
			Attributes: napi.Enumerable,
		})
	}
	if err := env.DefineProperties(exports, propDescs); err != nil {
		return nil, err
	}

	// Attach an object for "global" state to this instance of the module
	modData := moduleData{nil, newEnvStack()}
	if err := napi.SetInstanceData(env, &modData, moduleTeardown); err != nil {
		return nil, err
	}

	// Create a reference to the Template class and save it in the module data
	clsRef, err := env.CreateReference(propValues["Template"], 1)
	if err != nil {
		return nil, err
	}
	modData.templateConstructor = clsRef

	return exports, nil
}

func init() {
	napi.SetModuleInit(moduleInit)
}

func main() {}
