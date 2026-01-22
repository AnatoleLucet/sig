package internal

import (
	"iter"
	"reflect"
	"sync"
)

type Signal struct {
	*ReactiveNode

	mu           sync.RWMutex
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
	GetRuntime().tracker.Track(s)

	return s.Value()
}

func (s *Signal) Write(v any) {
	s.mu.Lock()
	if isEqual(s.valueUnsafe(), v) {
		s.mu.Unlock()
		return
	}

	s.pendingValue = &v
	s.mu.Unlock()

	r := GetRuntime()

	r.mu.Lock()
	s.SetVersion(r.scheduler.Time())
	r.heap.InsertAll(s.Subs())
	r.mu.Unlock()

	r.Schedule(true)
}

func (s *Signal) Value() any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.valueUnsafe()
}

func (s *Signal) valueUnsafe() any {
	if s.pendingValue != nil {
		return *s.pendingValue
	}
	return s.value
}

// Commit applies the pending value to the signal
func (s *Signal) Commit() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pendingValue != nil {
		s.value = *s.pendingValue
		s.pendingValue = nil
	}
}

// Subs returns an iterator over all subscribers
func (s *Signal) Subs() iter.Seq[*Computed] {
	return func(yield func(*Computed) bool) {
		s.mu.RLock()
		defer s.mu.RUnlock()
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
	s.mu.Lock()
	defer s.mu.Unlock()
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

func isEqual(a, b any) (result bool) {
	// todo: might want to make it configurable instead of always using reflect.DeepEqual

	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	aVal := reflect.ValueOf(a)
	bVal := reflect.ValueOf(b)

	if aVal.Type() != bVal.Type() {
		return false
	}

	if aVal.Type().Comparable() && bVal.Type().Comparable() {
		return a == b
	}

	return reflect.DeepEqual(a, b)
}
