//go:build wasm

package internal

import "sync"

var once sync.Once
var globalRuntime *Runtime

func GetRuntime() *Runtime {
	once.Do(func() {
		globalRuntime = NewRuntime()
	})

	return globalRuntime
}
