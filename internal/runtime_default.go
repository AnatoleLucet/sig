//go:build !wasm

package internal

import (
	"sync"

	"github.com/petermattis/goid"
)

var runtimes sync.Map

func GetRuntime() *Runtime {
	gid := getGID()

	if r, ok := runtimes.Load(gid); ok {
		return r.(*Runtime)
	}

	r := NewRuntime()
	runtimes.Store(gid, r)
	return r
}

func getGID() int64 {
	return goid.Get()
}
