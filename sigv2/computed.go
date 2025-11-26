package sigv2

// computedNode represents a reactive computation
type computedNode[T any] struct {
	*baseNode[T]
	fn     func(T) T
	equals EqualsFunc[T]
	name   string
}

// Ensure computedNode implements heapNode
var _ heapNode = (*computedNode[any])(nil)

// ComputedOptions configures computed behavior
type ComputedOptions[T any] struct {
	Name   string
	Equals EqualsFunc[T]
}

// newComputed creates a new computed value
func newComputed[T any](fn func(T) T, initial T, opts *ComputedOptions[T]) *computedNode[T] {
	node := &computedNode[T]{
		baseNode: &baseNode[T]{
			value:        initial,
			pendingValue: NotPending,
			time:         getClock(),
			statusFlags:  StatusUninitialized,
			parent:       getOwner(),
		},
		fn:   fn,
		name: "computed",
	}

	// Set self-referencing prevHeap
	node.baseNode.prevHeap = node.baseNode

	if opts != nil {
		if opts.Name != "" {
			node.name = opts.Name
		}
		node.equals = opts.Equals
	}

	// Default equality function - always recompute
	if node.equals == nil {
		node.equals = func(a, b T) bool { return false }
	}

	parent := getContext()
	if parent != nil {
		// Add to parent's child list
		if parentNode, ok := parent.(*computedNode[any]); ok {
			node.baseNode.nextSibling = parentNode.baseNode.firstChild
			parentNode.baseNode.firstChild = node.baseNode
		}

		// Set initial height and recompute
		if parent.getDepsTail() == nil {
			node.height = parent.getHeight()
			node.recompute(true)
		} else {
			node.height = parent.getHeight() + 1
			insertIntoHeap(node, getDirtyHeap())
		}
	} else {
		// No parent, compute immediately
		node.recompute(true)
	}

	return node
}

// Get reads the computed value
func (c *computedNode[T]) Get() T {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return readComputed(c)
}

// commitPendingValue implements Committable
func (c *computedNode[T]) commitPendingValue() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.pendingValue != NotPending {
		c.value = c.pendingValue.(T)
		c.pendingValue = NotPending
	}

	// Dispose children if this is a computed with a function
	if c.fn != nil {
		disposeChildren(c, true)
	}
}

// readComputed reads a computed value and tracks the dependency
func readComputed[T any](c *computedNode[T]) T {
	ctx := getContext()

	// Track dependency if we're in a reactive context
	if ctx != nil && isTracking() {
		linkNodes(c, ctx)

		// Check if we need to update this computed first
		isZombie := c.flags&FlagZombie != 0
		var minHeight int
		if isZombie {
			minHeight = getPendingHeap().min
		} else {
			minHeight = getDirtyHeap().min
		}

		if c.height >= minHeight {
			markNode(ctx, FlagDirty)
			if isZombie {
				markHeap(getPendingHeap(), func(n heapNode) {
					markNode(n, FlagDirty)
				})
			} else {
				markHeap(getDirtyHeap(), func(n heapNode) {
					markNode(n, FlagDirty)
				})
			}
			updateIfNecessary(c)
		}

		// Update subscriber height if needed
		height := c.height
		if height >= ctx.getHeight() {
			ctx.setHeight(height + 1)
		}
	}

	// Handle pending/error states
	if c.statusFlags&StatusPending != 0 {
		if (ctx != nil && !isStale()) || c.statusFlags&StatusUninitialized != 0 {
			panic(&NotReadyError{Cause: c})
		}
	}

	if c.statusFlags&StatusError != 0 {
		if c.time < getClock() {
			c.recompute(true)
			return readComputed(c)
		} else {
			panic(c.err)
		}
	}

	// Return pending or current value based on context
	if ctx == nil || c.pendingValue == NotPending {
		return c.value
	}

	if isStale() {
		return c.value
	}

	return c.pendingValue.(T)
}

// recompute updates a computed node's value (implements Recomputable)
func (el *computedNode[T]) recompute(create bool) {
	// Determine which heap to use
	var targetHeap *heap
	if el.flags&FlagZombie != 0 {
		targetHeap = getPendingHeap()
	} else {
		targetHeap = getDirtyHeap()
	}

	deleteFromHeap(el, targetHeap)

	// Handle pending disposal
	if el.pendingValue != NotPending || el.pendingFirstChild != nil || el.pendingDisposal != nil {
		disposeChildren(el, true)
	} else {
		// Mark for disposal
		markDisposal(el)
		getPendingNodes().add(el.baseNode)
		el.pendingDisposal = el.disposal
		el.pendingFirstChild = el.firstChild
		el.disposal = nil
		el.firstChild = nil
	}

	// Set up computation context
	oldContext := getContext()
	setContext(el)
	defer setContext(oldContext)

	el.depsTail = nil
	el.flags = FlagRecomputingDeps

	// Get current value
	value := el.value
	if el.pendingValue != NotPending {
		value = el.pendingValue.(T)
	}

	oldHeight := el.height
	el.time = getClock()
	prevStatusFlags := el.statusFlags
	prevError := el.err

	// Clear status flags before computation
	el.statusFlags = StatusNone
	el.err = nil

	// Execute computation
	defer func() {
		if r := recover(); r != nil {
			if notReady, ok := r.(*NotReadyError); ok {
				// Async operation - mark as pending
				el.statusFlags = (prevStatusFlags &^ StatusError) | StatusPending
				el.err = notReady
			} else {
				// Other error - mark as error
				el.statusFlags = StatusError | StatusUninitialized
				el.err = r.(error)
			}
		}

		el.flags = FlagNone
	}()

	value = el.fn(value)

	// Clean up old dependencies
	depsTail := el.depsTail
	var toRemove *link
	if depsTail != nil {
		toRemove = depsTail.nextDep
	} else {
		toRemove = el.deps
	}

	for toRemove != nil {
		toRemove = unlinkSubs(toRemove)
	}

	if depsTail != nil {
		depsTail.nextDep = nil
	} else {
		el.deps = nil
	}

	// Check if value or status changed
	valueChanged := !el.equals(el.value, value)
	if el.pendingValue != NotPending {
		valueChanged = !el.equals(el.pendingValue.(T), value)
	}

	statusFlagsChanged := el.statusFlags != prevStatusFlags || el.err != prevError

	if valueChanged || statusFlagsChanged {
		if valueChanged {
			if create {
				el.value = value
			} else {
				if el.pendingValue == NotPending {
					getPendingNodes().add(el)
				}
				el.pendingValue = value
			}
		}

		// Mark subscribers as dirty
		for s := el.subs; s != nil; s = s.nextSub {
			if sub, ok := s.sub.(heapNode); ok {
				var targetHeap *heap
				if sub.getFlags()&FlagZombie != 0 {
					targetHeap = getPendingHeap()
				} else {
					targetHeap = getDirtyHeap()
				}
				insertIntoHeap(sub, targetHeap)
			}
		}
	} else if el.height != oldHeight {
		// Height changed but value didn't
		for s := el.subs; s != nil; s = s.nextSub {
			if sub, ok := s.sub.(heapNode); ok {
				var targetHeap *heap
				if sub.getFlags()&FlagZombie != 0 {
					targetHeap = getPendingHeap()
				} else {
					targetHeap = getDirtyHeap()
				}
				insertIntoHeapHeight(sub, targetHeap)
			}
		}
	}
}

// updateIfNecessary checks and updates a computed node if needed
func updateIfNecessary[T any](el *computedNode[T]) {
	if el.flags&FlagCheck != 0 {
		// Check all dependencies
		for d := el.deps; d != nil; d = d.nextDep {
			dep := d.dep

			// If dependency is computed, check it first
			if comp, ok := dep.(heapNode); ok {
				if compNode, ok := comp.(*computedNode[any]); ok {
					updateIfNecessary(compNode)
				}
			}

			// Break early if we're already dirty
			if el.flags&FlagDirty != 0 {
				break
			}
		}
	}

	if el.flags&FlagDirty != 0 {
		el.recompute(false)
	}

	el.flags = FlagNone
}

// markDisposal marks a node's children as zombies
func markDisposal[T any](el *computedNode[T]) {
	child := el.firstChild
	for child != nil {
		if childNode, ok := interface{}(child).(*baseNode[any]); ok {
			childNode.flags |= FlagZombie

			// Move from dirty to pending heap if needed
			if childNode.flags&FlagInHeap != 0 {
				if comp, ok := interface{}(childNode).(*computedNode[any]); ok {
					deleteFromHeap(comp, getDirtyHeap())
					insertIntoHeap(comp, getPendingHeap())
					markDisposal(comp)
				}
			}

			child = childNode.nextSibling
		} else {
			break
		}
	}
}

// disposeChildren disposes a node's children
func disposeChildren[T any](node *computedNode[T], zombie bool) {
	var childInterface interface{}
	if zombie {
		childInterface = node.pendingFirstChild
	} else {
		childInterface = node.firstChild
	}

	for childInterface != nil {
		// Try to cast to computed node to access deps
		if comp, ok := childInterface.(*computedNode[any]); ok {
			nextChild := comp.baseNode.nextSibling

			if comp.deps != nil {
				// Determine heap
				var targetHeap *heap
				if comp.flags&FlagZombie != 0 {
					targetHeap = getPendingHeap()
				} else {
					targetHeap = getDirtyHeap()
				}

				deleteFromHeap(comp, targetHeap)

				// Remove all dependencies
				toRemove := comp.deps
				for toRemove != nil {
					toRemove = unlinkSubs(toRemove)
				}

				comp.deps = nil
				comp.depsTail = nil
				comp.flags = FlagNone
			}

			// Recursively dispose children
			disposeChildren(comp, zombie)

			childInterface = nextChild
		} else {
			break
		}
	}

	if zombie {
		node.pendingFirstChild = nil
	} else {
		node.firstChild = nil
		node.nextSibling = nil
	}

	// Run disposal functions
	var disposal interface{}
	if zombie {
		disposal = node.pendingDisposal
	} else {
		disposal = node.disposal
	}
	runDisposal(disposal)

	if zombie {
		node.pendingDisposal = nil
	} else {
		node.disposal = nil
	}
}
