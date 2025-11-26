package sigv2

// Signal creates a new reactive signal value
// Returns getter and setter functions following the sig/ pattern
func Signal[T comparable](initial T, opts ...SignalOptions[T]) (func() T, func(T)) {
	var options *SignalOptions[T]
	if len(opts) > 0 {
		options = &opts[0]
	}

	s := newSignal(initial, options)
	return s.Get, s.Set
}

// Computed creates a reactive computed value
// Returns a getter function that recomputes when dependencies change
func Computed[T any](fn func() T, opts ...ComputedOptions[T]) func() T {
	var options *ComputedOptions[T]
	if len(opts) > 0 {
		options = &opts[0]
	}

	// Wrap function to take previous value parameter
	var zero T
	wrappedFn := func(prev T) T {
		return fn()
	}

	node := newComputed(wrappedFn, zero, options)
	return node.Get
}

// Memo is an alias for Computed for compatibility with Solid.js naming
func Memo[T any](fn func() T, opts ...ComputedOptions[T]) func() T {
	return Computed(fn, opts...)
}

// Async creates an async computed value that handles promises and async iterators
// Returns a getter function and a refresh function
func Async[T any](fn func(prev T, refreshing bool) AsyncResult[T], initial T, opts ...ComputedOptions[T]) (func() T, func()) {
	var options *ComputedOptions[T]
	if len(opts) > 0 {
		options = &opts[0]
	}

	node := newAsyncComputed(fn, initial, options)
	return node.Get, node.Refresh
}

// Effect creates a side effect that runs when dependencies change
// The effect function receives the computed value and previous value
// It can optionally return a cleanup function
func Effect[T any](
	compute func() T,
	effect func(value T, prevValue T) DisposalFunc,
	opts ...EffectOptions[T],
) {
	var options *EffectOptions[T]
	if len(opts) > 0 {
		options = &opts[0]
	}

	// Wrap compute to take previous value
	var zero T
	wrappedCompute := func(prev T) T {
		return compute()
	}

	newEffect(wrappedCompute, effect, nil, zero, options)
}

// RenderEffect creates a render effect (runs during render phase)
func RenderEffect[T any](
	compute func() T,
	effect func(value T, prevValue T) DisposalFunc,
	opts ...EffectOptions[T],
) {
	var options *EffectOptions[T]
	if len(opts) > 0 {
		options = &opts[0]
	} else {
		options = &EffectOptions[T]{}
	}

	options.Render = true

	var zero T
	wrappedCompute := func(prev T) T {
		return compute()
	}

	newEffect(wrappedCompute, effect, nil, zero, options)
}

// Root creates a new reactive root scope
// The function receives a dispose callback to clean up the scope
func Root(fn func(dispose func())) interface{} {
	return createRoot(fn, nil)
}

// Batch executes multiple updates in a single batch
// All signal updates inside the function are queued and processed together
func Batch[T any](fn func() T) T {
	// Mark that we're in a batch
	batchDepth := incrementBatchDepth()
	defer decrementBatchDepth()

	result := fn()

	// If we're exiting the outermost batch and updates were queued, schedule flush
	if batchDepth == 1 {
		schedulerMutex.Lock()
		if scheduled {
			schedulerMutex.Unlock()
			// Already scheduled, let it run
		} else if getPendingNodes().len() > 0 || getDirtyHeap().max > 0 {
			// We have pending work, schedule it
			scheduled = true
			schedulerMutex.Unlock()
			go flush()
		} else {
			schedulerMutex.Unlock()
		}
	}

	return result
}

// Untrack executes a function without tracking dependencies
func Untrack[T any](fn func() T) T {
	return untrack(fn)
}

// OnCleanup registers a cleanup function for the current scope
func OnCleanup(fn DisposalFunc) {
	onCleanup(fn)
}

// GetOwner returns the current reactive owner
func GetOwner() *ownerNode {
	return getOwner()
}

// RunWithOwner executes a function with a specific owner context
func RunWithOwner(owner *ownerNode, fn func() interface{}) interface{} {
	return runWithOwner(owner, fn)
}

// WaitForFlush blocks until all pending reactive updates have been processed
// This is useful in tests or when you need to ensure updates are complete
func WaitForFlush() {
	waitForFlush()
}
