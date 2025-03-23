//go:build coverage

package napi

// #include <stdlib.h>
// void exitHook();
import "C"

//go:linkname runtime_runExitHooks internal/runtime/exithook.Run
func runtime_runExitHooks(exitCode int)

//export exitHook
func exitHook() {
	runtime_runExitHooks(0)
}

func init() {
	C.atexit((*[0]byte)(C.exitHook))
}
