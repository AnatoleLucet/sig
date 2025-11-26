package sigv2

import (
	"sync"

	"github.com/petermattis/goid"
)

// ownerNode represents a reactive scope that owns computations and effects
type ownerNode struct {
	mu sync.RWMutex

	root           bool
	parentComputed Computable
	disposal       interface{} // DisposalFunc or []DisposalFunc
	queue          *Queue
	contextMap     map[interface{}]interface{}
	parent         *ownerNode
	firstChild     interface{} // Can be ownerNode or computed node
	nextSibling    interface{}

	pendingDisposal   interface{}
	pendingFirstChild interface{}
}

// Computable interface implementations for ownerNode
func (o *ownerNode) getDeps() *link {
	return nil
}

func (o *ownerNode) getDepsTail() *link {
	return nil
}

func (o *ownerNode) setDeps(l *link) {
	// Owner nodes don't have dependencies
}

func (o *ownerNode) setDepsTail(l *link) {
	// Owner nodes don't have dependencies
}

func (o *ownerNode) getHeight() int {
	return 0
}

func (o *ownerNode) setHeight(h int) {
	// Owner nodes don't have height
}

func (o *ownerNode) getFlags() ReactiveFlags {
	return FlagNone
}

func (o *ownerNode) setFlags(f ReactiveFlags) {
	// Owner nodes don't have flags
}

func (o *ownerNode) getMutex() *sync.RWMutex {
	return &o.mu
}

// Global context tracking using goroutine IDs
var activeContexts sync.Map // map[int64]Computable

// getContext returns the currently active reactive context
func getContext() Computable {
	gid := goid.Get()
	if ctx, ok := activeContexts.Load(gid); ok {
		return ctx.(Computable)
	}
	return nil
}

// setContext sets the currently active reactive context
func setContext(ctx Computable) {
	gid := goid.Get()
	if ctx == nil {
		activeContexts.Delete(gid)
	} else {
		activeContexts.Store(gid, ctx)
	}
}

// getOwner returns the currently active owner
func getOwner() *ownerNode {
	ctx := getContext()
	if ctx == nil {
		return nil
	}

	// If context is a computed node, get its parent owner
	if node, ok := ctx.(*computedNode[any]); ok {
		return node.parent
	}

	// If context is an owner node
	if owner, ok := ctx.(*ownerNode); ok {
		return owner
	}

	return nil
}

// createRoot creates a new reactive root scope
func createRoot(fn func(dispose func()), opts *RootOptions) interface{} {
	parent := getOwner()

	owner := &ownerNode{
		root:       true,
		queue:      getGlobalQueue(),
		contextMap: make(map[interface{}]interface{}),
		parent:     parent,
	}

	if parent != nil {
		owner.parentComputed = parent.parentComputed
		owner.queue = parent.queue
		// Copy parent's context
		for k, v := range parent.contextMap {
			owner.contextMap[k] = v
		}

		// Add to parent's child list
		owner.nextSibling = parent.firstChild
		parent.firstChild = owner
	}

	return runWithOwner(owner, func() interface{} {
		if fn != nil {
			disposeFn := func() {
				disposeOwner(owner, false)
			}
			fn(disposeFn)
		}
		return nil
	})
}

// runWithOwner executes a function with a specific owner as context
func runWithOwner(owner *ownerNode, fn func() interface{}) interface{} {
	oldContext := getContext()
	setContext(owner)
	defer setContext(oldContext)

	return fn()
}

// onCleanup registers a cleanup function to run when the current scope is disposed
func onCleanup(fn DisposalFunc) DisposalFunc {
	owner := getOwner()
	if owner == nil {
		return fn
	}

	owner.mu.Lock()
	defer owner.mu.Unlock()

	if owner.disposal == nil {
		owner.disposal = fn
	} else if arr, ok := owner.disposal.([]DisposalFunc); ok {
		owner.disposal = append(arr, fn)
	} else {
		owner.disposal = []DisposalFunc{owner.disposal.(DisposalFunc), fn}
	}

	return fn
}

// disposeOwner disposes an owner and all its children
func disposeOwner(owner *ownerNode, zombie bool) {
	var disposal interface{}
	if zombie {
		disposal = owner.pendingDisposal
	} else {
		disposal = owner.disposal
	}

	runDisposal(disposal)

	if zombie {
		owner.pendingDisposal = nil
	} else {
		owner.disposal = nil
	}
}

// runDisposal executes disposal functions
func runDisposal(disposal interface{}) {
	if disposal == nil || disposal == NotPending {
		return
	}

	if arr, ok := disposal.([]DisposalFunc); ok {
		for _, fn := range arr {
			fn()
		}
	} else if fn, ok := disposal.(DisposalFunc); ok {
		fn()
	}
}

// RootOptions configures root scope behavior
type RootOptions struct {
	Queue *Queue
}

// untrack executes a function without tracking dependencies
func untrack[T any](fn func() T) T {
	tracking := getTracking()
	setTracking(false)
	defer setTracking(tracking)

	return fn()
}

var trackingEnabled sync.Map // map[int64]bool

// isTracking returns whether dependency tracking is currently enabled
func isTracking() bool {
	gid := goid.Get()
	if enabled, ok := trackingEnabled.Load(gid); ok {
		return enabled.(bool)
	}
	return true // Default to enabled
}

// setTracking sets whether dependency tracking is enabled
func setTracking(enabled bool) {
	gid := goid.Get()
	trackingEnabled.Store(gid, enabled)
}

// getTracking returns the current tracking state
func getTracking() bool {
	return isTracking()
}
