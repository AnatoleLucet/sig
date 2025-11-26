package sigv2

import "sync"

// ReactiveFlags represents the state of a reactive node in the dependency graph
type ReactiveFlags uint8

const (
	FlagNone ReactiveFlags = 0
	// FlagCheck indicates the node needs to check if dependencies changed
	FlagCheck ReactiveFlags = 1 << iota
	// FlagDirty indicates the node needs recomputation
	FlagDirty
	// FlagRecomputingDeps indicates the node is currently recomputing
	FlagRecomputingDeps
	// FlagInHeap indicates the node is in the dirty/pending heap
	FlagInHeap
	// FlagInHeapHeight indicates the node is in the heap for height adjustment
	FlagInHeapHeight
	// FlagZombie indicates the node is marked for disposal
	FlagZombie
)

// StatusFlags represents async state of a signal/computed
type StatusFlags uint8

const (
	StatusNone StatusFlags = 0
	// StatusPending indicates an async operation is in progress
	StatusPending StatusFlags = 1 << iota
	// StatusError indicates an error occurred
	StatusError
	// StatusUninitialized indicates the value hasn't been computed yet
	StatusUninitialized
)

// EffectType distinguishes between render and user effects
type EffectType uint8

const (
	EffectTypeRender EffectType = 1
	EffectTypeUser   EffectType = 2
)

// NotPending is a sentinel value indicating no pending update
var NotPending = &struct{ sentinel bool }{true}

// NotReadyError is thrown when an async signal is read before it's ready
type NotReadyError struct {
	Cause interface{}
}

func (e *NotReadyError) Error() string {
	return "async signal not ready"
}

// NoOwnerError is thrown when context is accessed without a reactive owner
type NoOwnerError struct{}

func (e *NoOwnerError) Error() string {
	return "Context can only be accessed under a reactive root."
}

// ContextNotFoundError is thrown when a context value is not found
type ContextNotFoundError struct{}

func (e *ContextNotFoundError) Error() string {
	return "Context must either be created with a default value or a value must be provided before accessing it."
}

// SupportsProxy indicates if the runtime supports proxy-like behavior
const SupportsProxy = true

// link represents a bidirectional link between a dependency and a subscriber
type link struct {
	dep     Observable
	sub     Computable
	nextDep *link
	prevSub *link
	nextSub *link
}

// Observable is anything that can be observed (signals, computed)
type Observable interface {
	getSubs() *link
	getSubsTail() *link
	setSubs(*link)
	setSubsTail(*link)
	getTime() uint64
	getPendingValue() interface{}
	getStatusFlags() StatusFlags
	getError() error
	getMutex() *sync.RWMutex
}

// Computable is anything that can compute/react (computed, effects)
type Computable interface {
	getDeps() *link
	getDepsTail() *link
	setDeps(*link)
	setDepsTail(*link)
	getHeight() int
	setHeight(int)
	getFlags() ReactiveFlags
	setFlags(ReactiveFlags)
	getMutex() *sync.RWMutex
}

// Disposable represents something that can be disposed
type Disposable interface {
	Dispose()
}

// DisposalFunc is a function that performs cleanup
type DisposalFunc func()

// EqualsFunc is a custom equality function
type EqualsFunc[T any] func(a, b T) bool
