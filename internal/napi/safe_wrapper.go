package napi

// #include <stdint.h>
import "C"
import (
	"fmt"
	"runtime/cgo"
	"unsafe"
)

type SafeWrapper[T any] struct {
	tag TypeTag
}

func NewSafeWrapper[T any](tag1 uint64, tag2 uint64) SafeWrapper[T] {
	return SafeWrapper[T]{TypeTag{C.uint64_t(tag1), C.uint64_t(tag2)}}
}

func (sfw *SafeWrapper[T]) Wrap(env Env, jsObject Value, goObject *T) error {
	// Wrap the object
	handle := cgo.NewHandle(goObject)
	goPtr, goCleanup := launderHandle(handle)
	var finalizeCleanup cleanupFunc
	finalizeCb, finalizePtr, finalizeCleanup := MakeNapiFinalize(func(env Env, data unsafe.Pointer) error {
		goCleanup()
		finalizeCleanup()
		return nil
	})
	err := env.Wrap(jsObject, goPtr, finalizeCb, finalizePtr)
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
		return nil, fmt.Errorf("tried to unwrap object with incorrect type tag")
	}

	// Unwrap the object
	wrapped, err := env.Unwrap(jsObject)
	if err != nil {
		return nil, err
	}
	handle := cgo.Handle(*(*uintptr)(wrapped))
	return handle.Value().(*T), nil
}
