package proto

// effect is a side-effect that runs when its dependencies change
type effect struct {
	nodeData

	computation func()
	cleanup     func()

	scheduler *Scheduler
}

func (e *effect) node() *nodeData {
	return &e.nodeData
}

func (e *effect) execute(sched *Scheduler) {
	// Remove from heap (lock held)
	sched.dirty.remove(e)

	// Run cleanup from previous execution
	if e.cleanup != nil {
		cleanup := e.cleanup
		e.cleanup = nil
		// Release lock before running cleanup
		sched.mu.Unlock()
		cleanup()
		sched.mu.Lock()
	}

	// Clear old dependencies
	toRemove := e.deps
	for toRemove != nil {
		toRemove = unlinkSub(toRemove)
	}
	e.deps = nil
	e.depsTail = nil

	// Mark as recomputing and track dependencies
	e.flags = FlagRecomputing

	prevReaction := sched.activeReaction
	sched.activeReaction = e

	// Release lock before running user code
	sched.mu.Unlock()

	// Run the effect (no lock held - user code can read signals/memos)
	e.computation()

	// Re-acquire lock to update state
	sched.mu.Lock()

	sched.activeReaction = prevReaction
	e.flags = FlagNone
}

// Dispose stops the effect and runs cleanup
func (e *effect) Dispose() {
	if e.cleanup != nil {
		e.cleanup()
		e.cleanup = nil
	}

	// Unlink from all dependencies
	toRemove := e.deps
	for toRemove != nil {
		toRemove = unlinkSub(toRemove)
	}
	e.deps = nil
	e.depsTail = nil
}

// Effect creates a side-effect with the default scheduler
func Effect(computation func()) *effect {
	return EffectWith(defaultScheduler, computation)
}

// EffectWith creates a side-effect with a specific scheduler
func EffectWith(sched *Scheduler, computation func()) *effect {
	e := &effect{
		nodeData: nodeData{
			height:   0,
			flags:    FlagDirty,
			heapPrev: nil,
		},
		computation: computation,
		scheduler:   sched,
	}
	e.heapPrev = e

	// Initial execution
	sched.mu.Lock()
	sched.dirty.insert(e)
	sched.mu.Unlock()
	sched.Stabilize()

	return e
}

// OnCleanup registers a cleanup function for the current effect
func OnCleanup(cleanup func()) {
	if e, ok := defaultScheduler.activeReaction.(*effect); ok {
		prevCleanup := e.cleanup
		if prevCleanup == nil {
			e.cleanup = cleanup
		} else {
			e.cleanup = func() {
				prevCleanup()
				cleanup()
			}
		}
	}
}
