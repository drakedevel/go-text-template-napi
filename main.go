package main

import (
	"fmt"

	"github.com/drakedevel/go-text-template-napi/internal/napi"
)

func moduleInit(env napi.Env, exports napi.Value) napi.Value {
	fmt.Printf("In N-API module Init\n")

	var propDescs []napi.PropertyDescriptor

	propDescs = append(propDescs, napi.PropertyDescriptor{
		Name:       env.CreateString("Template"),
		Value:      buildTemplateClass(env),
		Attributes: napi.Enumerable,
	})

	env.DefineProperties(exports, propDescs)

	return exports
}

func init() {
	napi.SetModuleInit(moduleInit)
}

func main() {}
