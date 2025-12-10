package internal

import (
	"sync"

	"github.com/petermattis/goid"
)

var runtimes sync.Map

func GetRuntime() *Runtime {
	gid := goid.Get()

	if rt, ok := runtimes.Load(gid); ok {
		return rt.(*Runtime)
	}

	rt := NewRuntime()
	runtimes.Store(gid, rt)
	return rt
}

type Runtime struct {
	mu sync.Mutex

	// incremented each time the scheduler is flushed (when reactive nodes are updated)
	// used for staleness detection
	clock int

	context *ExecutionContext
}

func NewRuntime() *Runtime {
	return &Runtime{
		clock: 0,

		context: NewContext(),
	}
}
