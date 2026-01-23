package internal

import (
	"sync"
)

type Runtime struct {
	mu sync.Mutex

	heap               *PriorityHeap
	tracker            *Tracker
	batcher            *Batcher
	scheduler          *Scheduler
	nodeQueue          *NodeQueue
	effectQueue        *EffectQueue
	settledQueue       *SettledQueue
	userSettledQueue   *SettledQueue
	renderSettledQueue *SettledQueue
}

func NewRuntime() *Runtime {
	return &Runtime{
		heap:               NewHeap(),
		tracker:            NewTracker(),
		batcher:            NewBatcher(),
		scheduler:          NewScheduler(),
		nodeQueue:          NewNodeQueue(),
		effectQueue:        NewEffectQueue(),
		settledQueue:       NewSettledQueue(),
		userSettledQueue:   NewSettledQueue(),
		renderSettledQueue: NewSettledQueue(),
	}
}

func (r *Runtime) Schedule(force bool) {
	// force basically means: dont reschedule if already running
	// this is used to avoid redundant flushes when scheduling from within a flush

	if !force && r.scheduler.IsRunning() {
		return
	}

	r.mu.Lock()
	r.scheduler.Schedule()
	shouldFlush := !r.batcher.IsBatching() && !r.scheduler.IsRunning()
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
		r.renderSettledQueue.Run()

		r.effectQueue.RunEffects(EffectUser)
		r.userSettledQueue.Run()

		// lock again in case effects scheduled more work
		r.mu.Lock()
	})

	if err != nil {
		panic(err)
	}

	r.settledQueue.Run()
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

func (r *Runtime) OnSettled(fn func()) {
	r.settledQueue.Enqueue(fn)
}

func (r *Runtime) OnUserSettled(fn func()) {
	r.userSettledQueue.Enqueue(fn)
}

func (r *Runtime) OnRenderSettled(fn func()) {
	r.renderSettledQueue.Enqueue(fn)
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
