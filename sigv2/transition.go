package sigv2

import "sync"

// transition tracks async operations across a reactive update
type transition struct {
	time         uint64
	pendingNodes []interface{} // Can be any node type
	asyncNodes   []interface{} // Nodes that threw NotReadyError
	queues       [2][]func(EffectType)
}

var (
	activeTransition *transition
	transitions      = make(map[*transition]bool)
	transitionMutex  sync.Mutex
)

// initTransition initializes or updates the active transition
func initTransition(node interface{}) {
	transitionMutex.Lock()
	defer transitionMutex.Unlock()

	currentClock := getClock()

	// Don't reinitialize if we already have an active transition for this clock
	if activeTransition != nil && activeTransition.time == currentClock {
		return
	}

	// Create new transition if none exists
	if activeTransition == nil {
		activeTransition = &transition{
			time:         currentClock,
			pendingNodes: make([]interface{}, 0),
			asyncNodes:   make([]interface{}, 0),
			queues:       [2][]func(EffectType){},
		}
	}

	transitions[activeTransition] = true
	activeTransition.time = currentClock

	// Move pending nodes to transition
	nodes := pendingNodesList.get()
	for _, n := range nodes {
		activeTransition.pendingNodes = append(activeTransition.pendingNodes, n)
	}
}

// addAsyncNode adds a node that threw NotReadyError to the transition
func addAsyncNode(node interface{}) {
	transitionMutex.Lock()
	defer transitionMutex.Unlock()

	if activeTransition == nil {
		return
	}

	// Check if already added
	for _, n := range activeTransition.asyncNodes {
		if n == node {
			return
		}
	}

	activeTransition.asyncNodes = append(activeTransition.asyncNodes, node)
}

// transitionComplete checks if all async operations in a transition have completed
func transitionComplete(t *transition) bool {
	transitionMutex.Lock()
	defer transitionMutex.Unlock()

	for _, node := range t.asyncNodes {
		// Check if node still has pending flag
		if base, ok := node.(*baseNode[any]); ok {
			base.mu.RLock()
			pending := base.statusFlags&StatusPending != 0
			base.mu.RUnlock()
			if pending {
				return false
			}
		}
	}

	return true
}

// clearActiveTransition removes the active transition
func clearActiveTransition() {
	transitionMutex.Lock()
	defer transitionMutex.Unlock()

	if activeTransition != nil {
		delete(transitions, activeTransition)
		activeTransition = nil
	}
}

// getActiveTransition returns the current active transition
func getActiveTransition() *transition {
	transitionMutex.Lock()
	defer transitionMutex.Unlock()
	return activeTransition
}

// AsyncResult represents a result that can be a channel or an iterator
type AsyncResult[T any] interface {
	// Resolve returns a channel that will emit the result or an error
	Resolve() <-chan Result[T]
}

// Result wraps a value or error
type Result[T any] struct {
	Value T
	Err   error
}

// ChannelResult wraps a channel as an AsyncResult
type ChannelResult[T any] struct {
	ch <-chan Result[T]
}

func (c *ChannelResult[T]) Resolve() <-chan Result[T] {
	return c.ch
}

// PromiseFunc is a function that resolves with a value or error
type PromiseFunc[T any] func() (T, error)

// Promise wraps a function as an AsyncResult
type Promise[T any] struct {
	fn PromiseFunc[T]
}

func (p *Promise[T]) Resolve() <-chan Result[T] {
	ch := make(chan Result[T], 1)
	go func() {
		value, err := p.fn()
		ch <- Result[T]{Value: value, Err: err}
		close(ch)
	}()
	return ch
}

// asyncComputedNode represents an async reactive computation
type asyncComputedNode[T any] struct {
	*computedNode[T]
	lastResult    interface{}
	refreshing    bool
	asyncFn       func(T, bool) AsyncResult[T]
	resultChannel <-chan Result[T]
}

// newAsyncComputed creates a new async computed value
func newAsyncComputed[T any](asyncFn func(T, bool) AsyncResult[T], initial T, opts *ComputedOptions[T]) *asyncComputedNode[T] {
	var ac *asyncComputedNode[T]

	// Wrapper function that handles async results
	fn := func(prev T) T {
		result := asyncFn(prev, ac.refreshing)
		ac.refreshing = false
		ac.lastResult = result

		// Get the channel
		ch := result.Resolve()
		ac.resultChannel = ch

		// Start goroutine to handle results
		go func() {
			for res := range ch {
				if ac.lastResult != result {
					return
				}

				// Update signal value
				ac.mu.Lock()
				if res.Err != nil {
					ac.statusFlags = StatusError
					ac.err = res.Err
				} else {
					ac.value = res.Value
					ac.statusFlags = StatusNone
					ac.err = nil
				}
				ac.time = getClock()
				ac.mu.Unlock()

				// Mark subscribers as dirty
				for l := ac.subs; l != nil; l = l.nextSub {
					if sub, ok := l.sub.(heapNode); ok {
						var heap *heap
						if sub.getFlags()&FlagZombie != 0 {
							heap = getPendingHeap()
						} else {
							heap = getDirtyHeap()
						}
						insertIntoHeap(sub, heap)
					}
				}

				schedule()
			}
		}()

		// Throw NotReadyError to suspend computation
		initTransition(ac)
		panic(&NotReadyError{Cause: ac})
	}

	ac = &asyncComputedNode[T]{
		computedNode: newComputed(fn, initial, opts),
		asyncFn:      asyncFn,
	}

	return ac
}

// Refresh forces a recomputation of the async computed
func (ac *asyncComputedNode[T]) Refresh() {
	ac.mu.Lock()
	ac.refreshing = true
	ac.mu.Unlock()

	ac.computedNode.recompute(false)
	flush()
}
