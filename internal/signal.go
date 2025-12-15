package internal

import "iter"

type Signal struct {
	*ReactiveNode

	value        any
	pendingValue *any // nil if no pending value

	subsHead *DependencyLink
}

func (r *Runtime) NewSignal(initial any) *Signal {
	s := &Signal{
		ReactiveNode: r.NewNode(),
		value:        initial,
	}

	return s
}

func (s *Signal) Read() any {
	r := GetRuntime()

	r.tracker.Track(s)

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

// Subs returns an iterator over all subscribers
func (s *Signal) Subs() iter.Seq[*Computed] {
	return func(yield func(*Computed) bool) {
		link := s.subsHead
		for link != nil {
			if !yield(link.sub) {
				return
			}

			link = link.nextSub
		}
	}
}

func (s *Signal) addSubLink(link *DependencyLink) {
	if s.subsHead == nil {
		s.subsHead = link
		link.prevSub = link // loop to self
		link.nextSub = nil
	} else {
		tail := s.subsHead.prevSub
		tail.nextSub = link
		link.prevSub = tail
		link.nextSub = nil
		s.subsHead.prevSub = link
	}
}

func (s *Signal) removeSubLink(link *DependencyLink) {
	// single node
	if link.prevSub == link {
		s.subsHead = nil
		link.prevSub = nil
		link.nextSub = nil
		return
	}

	// multiple nodes
	if link == s.subsHead {
		s.subsHead = link.nextSub
	} else {
		link.prevSub.nextSub = link.nextSub
	}

	if link.nextSub != nil {
		link.nextSub.prevSub = link.prevSub
	} else {
		s.subsHead.prevSub = link.prevSub
	}

	link.prevSub = nil
	link.nextSub = nil
}

func isEqual(a, b any) bool {
	return a == b
}
