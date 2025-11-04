package proto

// signal is a reactive value that can be read and written
type signal struct {
	nodeData

	value        any
	pendingValue *any

	scheduler *Scheduler
}

func (s *signal) node() *nodeData {
	return &s.nodeData
}

func (s *signal) read(sched *Scheduler) any {
	// Track dependency if we're in a reaction
	if sched.activeReaction != nil {
		sched.mu.Lock()
		linkDep(s, sched.activeReaction)
		sched.mu.Unlock()
	}

	// During stabilization, return pending value if we have one
	if s.pendingValue != nil {
		return *s.pendingValue
	}
	return s.value
}

func (s *signal) set(newValue any, sched *Scheduler) {
	sched.mu.Lock()

	// Check if value actually changed
	valueChanged := s.value != newValue
	if !valueChanged && s.pendingValue == nil {
		sched.mu.Unlock()
		return
	}

	// Buffer the change
	if valueChanged {
		if s.pendingValue == nil {
			sched.pendingNodes = append(sched.pendingNodes, s)
		}
		s.pendingValue = &newValue
	}

	// Mark all subscribers as dirty
	for l := s.subs; l != nil; l = l.nextSub {
		sched.dirty.insert(l.sub)
	}

	shouldStabilize := sched.batchDepth == 0
	sched.mu.Unlock()

	// Auto-stabilize if not batching
	if shouldStabilize {
		sched.Stabilize()
	}
}

// Signal creates a new signal with the default scheduler
func Signal(initial any) (get func() any, set func(any)) {
	return SignalWith(defaultScheduler, initial)
}

// SignalWith creates a new signal with a specific scheduler
func SignalWith(sched *Scheduler, initial any) (get func() any, set func(any)) {
	s := &signal{
		nodeData: nodeData{
			height:   0, // Signals are always at height 0
			heapPrev: nil,
		},
		value:     initial,
		scheduler: sched,
	}
	s.heapPrev = s

	return func() any {
			return s.read(sched)
		}, func(v any) {
			s.set(v, sched)
		}
}
