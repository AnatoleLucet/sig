package internal

import (
	"sync"

	"github.com/petermattis/goid"
)

var runtimes sync.Map

func GetRuntime() *Runtime {
	gid := goid.Get()

	if r, ok := runtimes.Load(gid); ok {
		return r.(*Runtime)
	}

	r := NewRuntime()
	runtimes.Store(gid, r)
	return r
}

type Runtime struct {
	heap        *PriorityHeap
	tracker     *Tracker
	batcher     *Batcher
	scheduler   *Scheduler
	nodeQueue   *NodeQueue
	effectQueue *EffectQueue
}

func NewRuntime() *Runtime {
	return &Runtime{
		heap:        NewHeap(),
		tracker:     NewTracker(),
		batcher:     NewBatcher(),
		scheduler:   NewScheduler(),
		nodeQueue:   NewNodeQueue(),
		effectQueue: NewEffectQueue(),
	}
}

func (r *Runtime) Schedule() {
	r.scheduler.Schedule()

	if !r.batcher.IsBatching() {
		r.Flush()
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
	// [ ] if height and value changed
	//     [x] insert subs in dirty heap

	if node.fn == nil {
		return
	}

	node.ClearDeps()
	node.SetVersion(r.scheduler.Time())

	r.tracker.RunWithNode(node, node.fn)

	// if node.MaxDepHeight() != oldHeight {
	r.heap.InsertAll(node.Subs())
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
