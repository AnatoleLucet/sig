package internal

import (
	"sync"
)

type Runtime struct {
	mu sync.Mutex

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
	r.mu.Lock()
	r.scheduler.Schedule()
	shouldFlush := !r.batcher.IsBatching()
	r.mu.Unlock()

	if shouldFlush {
		r.Flush()
	}
}

func (r *Runtime) Flush() {
	r.mu.Lock()
	defer r.mu.Unlock()

	err := r.scheduler.Run(func() {
		r.heap.Drain(r.recompute)

		r.nodeQueue.Commit()

		// unlock for effects to allow signal writes
		r.mu.Unlock()

		r.effectQueue.RunEffects(EffectRender)
		r.effectQueue.RunEffects(EffectUser)

		// lock again in case effects scheduled more work
		r.mu.Lock()
	})

	if err != nil {
		panic(err)
	}
}

func (r *Runtime) CurrentOwner() *Owner {
	return r.tracker.CurrentOwner()
}

func (r *Runtime) CurrentComputation() *Computed {
	return r.tracker.CurrentComputation()
}

func (r *Runtime) OnCleanup(fn func()) {
	owner := r.CurrentOwner()
	if owner != nil {
		owner.OnCleanup(fn)
	}
}

func (r *Runtime) recompute(node *Computed) {
	fn := node.getFn()
	if fn == nil {
		return
	}

	oldValue := node.Value()

	node.DisposeChildren()
	node.ClearDeps()
	node.SetVersion(r.scheduler.Time())

	r.tracker.RunWithComputation(node, fn)

	if !isEqual(oldValue, node.Value()) {
		r.heap.InsertAll(node.Subs())
	}
}
