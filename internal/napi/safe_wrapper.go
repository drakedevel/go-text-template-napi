package napi

// #include <stdint.h>
import "C"
import (
	"fmt"
	"runtime/cgo"
)

type SafeWrapper[T any] struct {
	tag TypeTag
}

func NewSafeWrapper[T any](tag1 uint64, tag2 uint64) SafeWrapper[T] {
	return SafeWrapper[T]{TypeTag{C.uint64_t(tag1), C.uint64_t(tag2)}}
}

func (sfw *SafeWrapper[T]) Wrap(env Env, jsObject Value, goObject *T, finalizeFunc finalizeFunc) error {
	// Wrap the object
	data, finalize, finalizeHint := makeDataAndFinalize(goObject, finalizeFunc)
	err := env.Wrap(jsObject, data, finalize, finalizeHint)
	if err != nil {
		return err
	}

	// Tag it so we can safely unwrap it
	return env.TypeTagObject(jsObject, &sfw.tag)
}

func (sfw *SafeWrapper[T]) Unwrap(env Env, jsObject Value) (*T, error) {
	// Check the type tag
	tagOk, err := env.CheckObjectTypeTag(jsObject, &sfw.tag)
	if err != nil {
		return nil, err
	}
	if !tagOk {
		return nil, fmt.Errorf("missing or invalid type tag")
	}

	// Unwrap the object
	wrapped, err := env.Unwrap(jsObject)
	if err != nil {
		return nil, err
	}
	handle := cgo.Handle(*(*uintptr)(wrapped))
	return handle.Value().(*T), nil
}
