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

	dirtyHeap   *PriorityHeap
	pendingHeap *PriorityHeap

	queue       *EffectQueue
	scheduler   *NodeScheduler
	execContext *ExecutionContext
}

func NewRuntime() *Runtime {
	return &Runtime{
		dirtyHeap:   NewHeap(),
		pendingHeap: NewHeap(),

		queue:       NewQueue(),
		scheduler:   NewScheduler(),
		execContext: NewContext(),
	}
}

func (r *Runtime) Flush() {
	r.scheduler.Run(func(commit func()) {
		r.dirtyHeap.Run(r.recompute)

		commit()

		r.queue.RunEffects(EffectRender)
		r.queue.RunEffects(EffectUser)
	})
}
