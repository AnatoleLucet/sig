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

func (r *Runtime) CurrentOwner() *Owner {
	return r.tracker.currentOwner
}

func (r *Runtime) CurrentComputation() *Computed {
	return r.tracker.currentComputation
}

func (r *Runtime) OnCleanup(fn func()) {
	owner := r.CurrentOwner()
	if owner != nil {
		owner.OnCleanup(fn)
	}
}

func (r *Runtime) recompute(node *Computed) {
	if node.fn == nil {
		return
	}

	node.DisposeChildren()

	node.ClearDeps()
	node.SetVersion(r.scheduler.Time())

	r.tracker.RunWithComputation(node, node.fn)

	// todo: only do this if height and value changed
	r.heap.InsertAll(node.Subs())
}
