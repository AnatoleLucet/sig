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

	if r.context.ShouldTrack() {
		r.context.currentNode.Link(s.ReactiveNode)
	}

	return s.Value()
}

func (s *Signal) Write(v any) {
	// [ ] check if value changed
	// [x] set pending value
	// [ ] udpate time
	// [ ] insert subs in dirty heap
	// [ ] schedule node

	s.pendingValue = &v
}

func (s *Signal) Value() any {
	if s.pendingValue != nil {
		return *s.pendingValue
	}

	return s.value
}
