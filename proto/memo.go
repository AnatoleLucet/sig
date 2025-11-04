package proto

// memo is a computed value that caches its result
type memo struct {
	nodeData

	computation  func() any
	value        any
	pendingValue *any

	scheduler *Scheduler
}

func (m *memo) node() *nodeData {
	return &m.nodeData
}

func (m *memo) read(sched *Scheduler) any {
	// Track dependency if we're in a reaction
	if sched.activeReaction != nil {
		sched.mu.Lock()
		linkDep(m, sched.activeReaction)

		// Adjust consumer's height if needed
		consumerNode := sched.activeReaction.node()
		if m.height >= consumerNode.height {
			consumerNode.height = m.height + 1
		}
		sched.mu.Unlock()
	}

	// Return pending value if we have one, otherwise committed value
	// Note: pending values are only set during stabilization, so this is safe without lock
	if m.pendingValue != nil {
		return *m.pendingValue
	}
	return m.value
}

func (m *memo) execute(sched *Scheduler) {
	// Remove from heap and prepare for recomputation (lock held)
	sched.dirty.remove(m)

	// Clear old dependencies
	toRemove := m.deps
	for toRemove != nil {
		toRemove = unlinkSub(toRemove)
	}
	m.deps = nil
	m.depsTail = nil

	// Mark as recomputing and track dependencies
	m.flags = FlagRecomputing
	oldHeight := m.height

	prevReaction := sched.activeReaction
	sched.activeReaction = m

	// Release lock before running user code
	sched.mu.Unlock()

	// Compute new value (no lock held - user code can read other signals/memos)
	newValue := m.computation()

	// Re-acquire lock to update state
	sched.mu.Lock()

	sched.activeReaction = prevReaction
	m.flags = FlagNone

	// Check if value changed
	var valueChanged bool
	if m.pendingValue != nil {
		valueChanged = *m.pendingValue != newValue
	} else {
		valueChanged = m.value != newValue
	}

	// Buffer the change
	if valueChanged {
		if m.pendingValue == nil {
			sched.pendingNodes = append(sched.pendingNodes, m)
		}
		m.pendingValue = &newValue

		// Mark subscribers as dirty
		for l := m.subs; l != nil; l = l.nextSub {
			sched.dirty.insert(l.sub)
		}
	} else if m.height != oldHeight {
		// Height changed but value didn't - still need to update subscribers
		for l := m.subs; l != nil; l = l.nextSub {
			sched.dirty.insertAtHeight(l.sub)
		}
	}
}

// Memo creates a computed value with the default scheduler
func Memo(computation func() any) func() any {
	return MemoWith(defaultScheduler, computation)
}

// MemoWith creates a computed value with a specific scheduler
func MemoWith(sched *Scheduler, computation func() any) func() any {
	m := &memo{
		nodeData: nodeData{
			height:   0,
			flags:    FlagDirty,
			heapPrev: nil,
		},
		computation: computation,
		scheduler:   sched,
	}
	m.heapPrev = m

	// Initial computation
	sched.mu.Lock()
	sched.dirty.insert(m)
	sched.mu.Unlock()
	sched.Stabilize()

	return func() any {
		return m.read(sched)
	}
}
