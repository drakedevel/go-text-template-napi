package main

import (
	"fmt"

	"github.com/drakedevel/go-text-template-napi/internal/napi"
)

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

	return exports, nil
}

func init() {
	napi.SetModuleInit(moduleInit)
}

func main() {}
