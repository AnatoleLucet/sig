//go:build wasm

package internal

import (
	"bytes"
	"runtime"
	"strconv"
	"sync"
)

var runtimes sync.Map

func GetRuntime() *Runtime {
	gid := getGoroutineID()

	if r, ok := runtimes.Load(gid); ok {
		return r.(*Runtime)
	}

	r := NewRuntime()
	runtimes.Store(gid, r)
	return r
}

func getGoroutineID() int64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := bytes.Fields(buf[:n])[1]
	id, _ := strconv.ParseInt(string(idField), 10, 64)
	return id
}
