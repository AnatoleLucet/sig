package sigv2

import "sync"

var (
	pendingValueCheck sync.Map // map[int64]bool - per goroutine
	pendingCheck      sync.Map // map[int64]*pendingCheckState - per goroutine
)

type pendingCheckState struct {
	value bool
}

// Pending reads the pending (optimistic) value of signals/computed
// This allows reading values that haven't been committed yet
func Pending[T any](fn func() T) T {
	gid := getGoroutineID()
	prevLatest, _ := pendingValueCheck.LoadOrStore(gid, false)
	pendingValueCheck.Store(gid, true)
	defer pendingValueCheck.Store(gid, prevLatest)

	return staleValues(fn, false)
}

// IsPending checks if any signals/computed in the function are pending
func IsPending(fn func(), loadingValue ...bool) bool {
	gid := getGoroutineID()

	current, _ := pendingCheck.Load(gid)

	state := &pendingCheckState{value: false}
	pendingCheck.Store(gid, state)

	defer func() {
		if current != nil {
			pendingCheck.Store(gid, current)
		} else {
			pendingCheck.Delete(gid)
		}
	}()

	// Try to execute the function
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(*NotReadyError); !ok {
				// Return false for non-NotReadyError panics
				if len(loadingValue) > 0 {
					state.value = loadingValue[0]
				} else {
					state.value = false
				}
			} else {
				// NotReadyError means we're pending
				if len(loadingValue) > 0 {
					state.value = loadingValue[0]
				} else {
					// Re-panic if no loading value provided
					panic(r)
				}
			}
		}
	}()

	staleValues(func() interface{} {
		fn()
		return nil
	}, true)

	return state.value
}

// StaleValues executes a function while reading stale (current) values
// even if there are pending updates
func StaleValues[T any](fn func() T) T {
	return staleValues(fn, true)
}

// createOptimistic creates an optimistic signal that updates immediately
// This is used internally for optimistic UI updates
func createOptimistic[T comparable](initial T) *signal[T] {
	s := newSignal(initial, nil)

	// Mark as optimistic - values update immediately without batching
	// We'll add this field to signal struct
	return s
}

// isPendingValueCheck returns whether we're currently checking pending values
func isPendingValueCheck() bool {
	gid := getGoroutineID()
	if check, ok := pendingValueCheck.Load(gid); ok {
		return check.(bool)
	}
	return false
}

// isPendingCheck returns the current pending check state
func isPendingCheck() *pendingCheckState {
	gid := getGoroutineID()
	if state, ok := pendingCheck.Load(gid); ok {
		return state.(*pendingCheckState)
	}
	return nil
}

// Resolve waits for an async signal to resolve and returns its value
func Resolve[T any](fn func() T) <-chan T {
	ch := make(chan T, 1)

	go func() {
		defer close(ch)

		// Create a root to track the computation
		createRoot(func(dispose func()) {
			defer dispose()

			// Create a computed that tries to read the value
			node := newComputed(func(_ interface{}) interface{} {
				defer func() {
					if r := recover(); r != nil {
						if _, ok := r.(*NotReadyError); ok {
							// Still pending, will retry on next update
							return
						}
						// Other error, don't handle
						panic(r)
					}
				}()

				value := fn()
				ch <- value
				dispose()

				return value
			}, nil, nil)

			// Trigger initial computation
			_ = node
		}, nil)
	}()

	return ch
}
