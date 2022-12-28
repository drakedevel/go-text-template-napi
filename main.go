package main

import (
	"fmt"

	"github.com/drakedevel/go-text-template-napi/internal/napi"
)

type moduleData struct {
	templateConstructor napi.Ref
}

func getInstanceData(env napi.Env) (*moduleData, error) {
	raw, err := napi.GetInstanceData(env)
	if err != nil {
		return nil, err
	}
	return raw.(*moduleData), nil
}

func moduleInit(env napi.Env, exports napi.Value) (napi.Value, error) {
	fmt.Printf("In N-API module Init\n")

	var propDescs []napi.PropertyDescriptor

	clsName, err := env.CreateString("Template")
	if err != nil {
		return nil, err
	}
	clsValue, err := buildTemplateClass(env)
	if err != nil {
		return nil, err
	}
	propDescs = append(propDescs, napi.PropertyDescriptor{
		Name:       clsName,
		Value:      clsValue,
		Attributes: napi.Enumerable,
	})

	if err := env.DefineProperties(exports, propDescs); err != nil {
		return nil, err
	}

	// Create a reference to the Template class and save it in the module data
	// TODO: Don't leak clsRef
	clsRef, err := env.CreateReference(clsValue, 1)
	if err != nil {
		return nil, err
	}
	modData := moduleData{clsRef}
	if err := napi.SetInstanceData(env, &modData); err != nil {
		return nil, err
	}

	return exports, nil
}

func init() {
	napi.SetModuleInit(moduleInit)
}

func main() {}
