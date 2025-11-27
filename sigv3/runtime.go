package sig

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

	dirtyHeap   *Heap
	pendingHeap *Heap

	clock     int
	scheduled bool

	tracking       bool
	currentContext *ReactiveNode
}

func NewRuntime() *Runtime {
	return &Runtime{
		dirtyHeap:   NewHeap(),
		pendingHeap: NewHeap(),

		clock:     0,
		scheduled: false,

		tracking:       false,
		currentContext: nil,
	}
}

func (r *Runtime) Flush() {
	r.dirtyHeap.Run(func(node *ReactiveNode) {
		// recompute
		// or
		// adjust height
	})

	// commit pending values
	// for _, node := range q.pendingNodes {
	// 	if node.pendingValue != nil {
	// 		node.value = *node.pendingValue
	// 		node.pendingValue = nil
	// 	}
	// }
}
