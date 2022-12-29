package main

import (
	"fmt"

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
	fmt.Println("In N-API module teardown")
	modData := data.(*moduleData)
	if err := env.DeleteReference(modData.templateConstructor); err != nil {
		return err
	}
	return nil
}

func moduleInit(env napi.Env, exports napi.Value) (napi.Value, error) {
	fmt.Println("In N-API module init")

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

	// Attach an object for "global" state to this instance of the module
	modData := moduleData{nil, newEnvStack()}
	if err := napi.SetInstanceData(env, &modData, moduleTeardown); err != nil {
		return nil, err
	}

	// Create a reference to the Template class and save it in the module data
	clsRef, err := env.CreateReference(clsValue, 1)
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
