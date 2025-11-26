package sigv2

// EffectFn is the function that runs when an effect executes
type EffectFn[T any] func(value T, prevValue T) DisposalFunc

// ErrorFn handles errors in effects
type ErrorFn func(err error, reset func())

// EffectOptions configures effect behavior
type EffectOptions[T any] struct {
	Name   string
	Defer  bool
	Render bool
}

// effectNode represents a side effect that runs when dependencies change
type effectNode[T any] struct {
	*computedNode[T]
	effectFn  EffectFn[T]
	errorFn   ErrorFn
	cleanup   DisposalFunc
	modified  bool
	prevValue T
	queue     *Queue
	effectType EffectType
}

// newEffect creates a new effect
func newEffect[T any](
	compute func(T) T,
	effectFn EffectFn[T],
	errorFn ErrorFn,
	initial T,
	opts *EffectOptions[T],
) *effectNode[T] {
	// Determine effect type
	effectType := EffectTypeUser
	if opts != nil && opts.Render {
		effectType = EffectTypeRender
	}

	owner := getOwner()
	queue := getGlobalQueue()
	if owner != nil {
		queue = owner.queue
	}

	var initialized bool

	effect := &effectNode[T]{
		effectFn:   effectFn,
		errorFn:    errorFn,
		prevValue:  initial,
		queue:      queue,
		effectType: effectType,
		modified:   true,
	}

	// Create custom equals function that tracks modifications and enqueues effect
	customOpts := &ComputedOptions[T]{
		Equals: func(prev, val T) bool {
			// Always consider changed for simplicity
			equal := false // In real impl, would use provided equals or compare values

			if initialized {
				effect.modified = !equal
				// Enqueue effect if value changed and no error
				if !equal {
					effect.mu.RLock()
					hasError := effect.statusFlags&StatusError != 0
					effect.mu.RUnlock()

					if !hasError {
						effect.queue.enqueue(effect.effectType, effect.runEffect)
					}
				}
			}

			return equal
		},
	}

	if opts != nil && opts.Name != "" {
		customOpts.Name = opts.Name
	}

	// Create underlying computed node
	effect.computedNode = newComputed(compute, initial, customOpts)

	// Mark as initialized after creation
	initialized = true

	// Wrap the compute function for render effects to use stale values
	if effectType == EffectTypeRender {
		originalFn := effect.computedNode.fn
		effect.computedNode.fn = func(p T) T {
			// Check if we have an error
			effect.mu.RLock()
			hasError := effect.statusFlags&StatusError != 0
			effect.mu.RUnlock()

			if !hasError {
				return staleValues(func() T {
					return originalFn(p)
				}, true)
			}
			return originalFn(p)
		}
	}

	// Queue initial effect execution if not deferred
	if opts == nil || !opts.Defer {
		effect.mu.RLock()
		hasError := effect.statusFlags&(StatusError|StatusPending) != 0
		effect.mu.RUnlock()

		if !hasError {
			if effectType == EffectTypeUser {
				queue.enqueue(effectType, effect.runEffect)
			} else {
				effect.runEffect(effectType)
			}
		}
	}

	// Register cleanup
	onCleanup(func() {
		if effect.cleanup != nil {
			effect.cleanup()
		}
	})

	return effect
}

// runEffect executes the effect function
func (e *effectNode[T]) runEffect(effectType EffectType) {
	e.mu.Lock()
	modified := e.modified
	e.mu.Unlock()

	if !modified {
		return
	}

	// Clean up previous effect
	if e.cleanup != nil {
		e.cleanup()
		e.cleanup = nil
	}

	e.mu.RLock()
	value := e.value
	prevValue := e.prevValue
	e.mu.RUnlock()

	// Execute effect function
	defer func() {
		if r := recover(); r != nil {
			// Notify queue of error
			e.queue.mu.Lock()
			defer e.queue.mu.Unlock()

			// Handle error based on effect type
			if e.effectType == EffectTypeUser {
				if e.errorFn != nil {
					resetFn := func() {
						if e.cleanup != nil {
							e.cleanup()
							e.cleanup = nil
						}
					}

					// Convert recovered value to error
					var err error
					if e, ok := r.(error); ok {
						err = e
					} else {
						err = &NotReadyError{Cause: r}
					}

					// Call error handler in safe context
					func() {
						defer func() {
							// Ignore panics in error handler
							recover()
						}()
						e.errorFn(err, resetFn)
					}()
				}
			}
		}

		e.mu.Lock()
		e.prevValue = value
		e.modified = false
		e.mu.Unlock()
	}()

	cleanup := e.effectFn(value, prevValue)
	e.mu.Lock()
	e.cleanup = cleanup
	e.mu.Unlock()
}
