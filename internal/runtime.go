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
	heap        *PriorityHeap
	context     *ExecutionContext
	scheduler   *Scheduler
	nodeQueue   *NodeQueue
	effectQueue *EffectQueue
}

func NewRuntime() *Runtime {
	return &Runtime{
		heap:        NewHeap(),
		context:     NewContext(),
		scheduler:   NewScheduler(),
		nodeQueue:   NewNodeQueue(),
		effectQueue: NewEffectQueue(),
	}
}

func (r *Runtime) Flush() {
	r.scheduler.Run(func() {
		r.heap.Drain(r.recompute)

		r.nodeQueue.Commit()

		r.effectQueue.RunEffects(EffectRender)
		r.effectQueue.RunEffects(EffectUser)
	})
}

func (r *Runtime) recompute(node *ReactiveNode) {
	// [x] clear deps
	// [x] if fn!=nil run with node in exec context
	// [x] if height and value changed, insert subs in dirty heap

	if node.fn == nil {
		return
	}

	node.ClearDeps()
	r.context.RunWithNode(node, node.fn)

	// if heightChanged  {
	for sub := range node.Subs() {
		r.heap.Insert(sub)
	}
	// }
}

// 0. setSignal
//    set pending value
//    insert subs in dirty heap
// 1. schedule
//    calls flush
// 2. flush
//    run dirty heap (recompute each node)
//    commit pending values
//    run render&user effect
