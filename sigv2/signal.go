package sigv2

// signal represents a reactive value that can be read and written
type signal[T comparable] struct {
	*baseNode[T]
	equals    EqualsFunc[T]
	pureWrite bool
	name      string
}

// SignalOptions configures signal behavior
type SignalOptions[T comparable] struct {
	Name      string
	Equals    EqualsFunc[T]
	PureWrite bool
}

// newSignal creates a new signal with the given initial value
func newSignal[T comparable](initial T, opts *SignalOptions[T]) *signal[T] {
	s := &signal[T]{
		baseNode: &baseNode[T]{
			value:        initial,
			pendingValue: NotPending,
			time:         getClock(),
			statusFlags:  StatusNone,
		},
		name: "signal",
	}

	// Set self-referencing prevHeap for circular list
	s.baseNode.prevHeap = s.baseNode

	if opts != nil {
		if opts.Name != "" {
			s.name = opts.Name
		}
		s.equals = opts.Equals
		s.pureWrite = opts.PureWrite
	}

	// Default equality function
	if s.equals == nil {
		s.equals = func(a, b T) bool { return a == b }
	}

	return s
}

// Get reads the signal's current value
func (s *signal[T]) Get() T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return readSignal[T](s)
}

// Set updates the signal's value
func (s *signal[T]) Set(newValue T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	setSignalValue(s, newValue)
}

// Update applies a function to transform the signal's value
func (s *signal[T]) Update(fn func(T) T) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentValue := s.value
	if s.pendingValue != NotPending {
		currentValue = s.pendingValue.(T)
	}

	newValue := fn(currentValue)
	setSignalValue(s, newValue)
}

// commitPendingValue implements Committable
func (s *signal[T]) commitPendingValue() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.pendingValue != NotPending {
		s.value = s.pendingValue.(T)
		s.pendingValue = NotPending
	}
}

// readSignal reads a signal's value and tracks the dependency
func readSignal[T comparable](s *signal[T]) T {
	ctx := getContext()

	// Track dependency if we're in a reactive context
	if ctx != nil && isTracking() {
		linkNodes(s, ctx)

		// Signals don't need updating (they're always up to date)
		// Just update subscriber height if needed
		height := s.height
		if height >= ctx.getHeight() {
			ctx.setHeight(height + 1)
		}
	}

	// Handle pending/error states
	if s.statusFlags&StatusPending != 0 {
		if (ctx != nil && !isStale()) || s.statusFlags&StatusUninitialized != 0 {
			panic(&NotReadyError{Cause: s})
		}
	}

	if s.statusFlags&StatusError != 0 {
		if s.time < getClock() {
			// Try to recompute on next clock
			// For signals this shouldn't happen, but keep for compatibility
			panic(s.err)
		} else {
			panic(s.err)
		}
	}

	// Return pending or current value based on context
	if s.pendingValue == NotPending {
		return s.value
	}

	// If no context or reading stale values, return current value
	if ctx == nil || isStale() {
		return s.value
	}

	return s.pendingValue.(T)
}

// setSignalValue updates a signal's value and notifies subscribers
func setSignalValue[T comparable](s *signal[T], newValue T) {
	currentValue := s.value
	if s.pendingValue != NotPending {
		currentValue = s.pendingValue.(T)
	}

	// Check if value actually changed
	valueChanged := !s.equals(currentValue, newValue)

	// Only proceed if value changed or there are status flags to clear
	if !valueChanged && s.statusFlags == StatusNone {
		return
	}

	if valueChanged {
		// Store as pending value
		if s.pendingValue == NotPending {
			getPendingNodes().add(s)
		}
		s.pendingValue = newValue
	}

	// Clear any async status flags
	s.statusFlags = StatusNone
	s.err = nil
	s.time = getClock()

	// Mark all subscribers as needing updates
	dirtyHeap := getDirtyHeap()
	pendingHeap := getPendingHeap()

	for l := s.subs; l != nil; l = l.nextSub {
		if sub, ok := l.sub.(heapNode); ok {
			// Check if subscriber is a zombie (being disposed)
			var targetHeap *heap
			if sub.getFlags()&FlagZombie != 0 {
				targetHeap = pendingHeap
			} else {
				targetHeap = dirtyHeap
			}
			insertIntoHeap(sub, targetHeap)
		}
	}

	// Schedule a flush if we have subscribers OR if value changed
	if s.subs != nil || valueChanged {
		schedule()
	}
}
