package sig

import "github.com/AnatoleLucet/sig/internal"

func as[T any](v any) T {
	if v == nil {
		var zero T
		return zero
	}

	return v.(T)
}

type Signal[T any] struct {
	signal *internal.Signal
}

// NewSignal creates your tipical read/write signal.
func NewSignal[T any](initial T) *Signal[T] {
	return &Signal[T]{
		internal.GetRuntime().NewSignal(initial),
	}
}

// Read the current value of the signal, tracking the dependency if within a reactive context.
func (s *Signal[T]) Read() T {
	return as[T](s.signal.Read())
}

// Write a new value to the signal, triggering updates to any dependents.
func (s *Signal[T]) Write(v T) {
	s.signal.Write(v)
}

type Computed[T any] struct {
	computed *internal.Computed
}

// NewComputed creates a computed signal that derives its value from other signals (its a memo).
func NewComputed[T any](compute func() T) *Computed[T] {
	return &Computed[T]{
		internal.GetRuntime().NewComputed(func(c *internal.Computed) any {
			return compute()
		}),
	}
}

// Read the current value of the computed signal, tracking the dependency if within a reactive context.
func (c *Computed[T]) Read() T {
	return as[T](c.computed.Signal.Read())
}

type AsyncComputed[T any] struct{}

// NewAsyncComputed not implemented yet.
func NewAsyncComputed[T any](fn func() (T, error)) *AsyncComputed[T] {
	return &AsyncComputed[T]{}
}

// Read the current value of the async computed signal, tracking the dependency if within a reactive context.
func (c *AsyncComputed[T]) Read() (T, error) {
	return *new(T), nil
}

// NewBatch batches multiple signal writes into a single update cycle,
// instead of triggering updates after each write.
func NewBatch(fn func()) {
	internal.GetRuntime().NewBatch(fn)
}

// NewEffect creates a reactive effect that runs the given function
// whenever its dependencies change.
func NewEffect(fn func()) {
	internal.GetRuntime().NewEffect(internal.EffectUser, fn)
}

// Untrack runs the given function without tracking any reactive dependencies.
func Untrack[T any](fn func() T) T {
	var result T
	internal.GetRuntime().Untrack(func() { result = fn() })
	return result
}

// IsPending not implemented yet.
func IsPending(fn func()) bool {
	return false
}

// OnCleanup registers a function to be called when the current owner is disposed.
func OnCleanup(fn func()) {
	internal.GetRuntime().OnCleanup(fn)
}

type Context[T any] struct {
	ctx *internal.Context
}

// NewContext creates a new reactive context with an initial value.
func NewContext[T any](initial T) *Context[T] {
	return &Context[T]{
		internal.GetRuntime().NewContext(initial),
	}
}

// Value retrieves the current value of the context,
// inheriting from parent owners if not set in the current owner.
func (c *Context[T]) Value() T {
	return as[T](c.ctx.Value())
}

// Set a new value for the context in the current owner.
func (c *Context[T]) Set(value T) {
	c.ctx.Set(value)
}

type Owner struct {
	owner *internal.Owner
}

// NewOwner creates a new reactive owner.
// An owner manages the lifecycle of reactive nodes created within its context.
func NewOwner() *Owner {
	return &Owner{
		internal.GetRuntime().NewOwner(),
	}
}

// Run a function within the context of this owner.
// Each reactive node created within the function will be a child of this owner,
// and will be disposed when owner.Dispose() is called on this owner.
func (o *Owner) Run(fn func() error) error { return o.owner.Run(fn) }

// Dispose this owner and all its children.
func (o *Owner) Dispose() { o.owner.Dispose() }

// Add a cleanup function to be called ONCE when the owner is disposed.
func (o *Owner) OnCleanup(fn func()) { o.owner.OnCleanup(fn) }

// Add a function to be called when the owner is disposed (each time Dispose is called).
func (o *Owner) OnDispose(fn func()) { o.owner.OnDispose(fn) }

// Add a function to be called when a panic occurs within this owner.
// If no error listener is registered, the panic will propagate as usual.
func (o *Owner) OnError(fn func(any)) { o.owner.OnError(fn) }
