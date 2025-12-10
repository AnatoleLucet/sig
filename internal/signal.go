package internal

type Signal struct {
	*ReactiveNode

	value        any
	pendingValue *any // nil if no pending value
}

func (r *Runtime) NewSignal(initial any) *Signal {
	s := &Signal{
		ReactiveNode: r.NewNode(),
		value:        initial,
	}

	s.fn = nil // signals don't recompute

	return s
}

func (s *Signal) Read() any {
	r := GetRuntime()

	if r.tracker.ShouldTrack() {
		r.tracker.currentNode.Link(s.ReactiveNode)
	}

	return s.Value()
}

func (s *Signal) Write(v any) {
	r := GetRuntime()
	// [x] check if value changed
	// [x] set pending value
	// [x] udpate node time
	// [x] insert subs in dirty heap
	// [x] schedule node

	if isEqual(s.Value(), v) {
		return
	}

	s.pendingValue = &v
	s.SetVersion(r.scheduler.Time())

	r.heap.InsertAll(s.Subs())
	r.Schedule()
}

func (s *Signal) Value() any {
	if s.pendingValue != nil {
		return *s.pendingValue
	}

	return s.value
}

// Commit applies the pending value to the signal
func (s *Signal) Commit() {
	if s.pendingValue != nil {
		s.value = *s.pendingValue
		s.pendingValue = nil
	}
}

func isEqual(a, b any) bool {
	return a == b
}
