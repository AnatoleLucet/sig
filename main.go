package sig

func Signal[T any](initial T) (func() T, func(T)) {
	value := initial

	get := func() T { return value }
	set := func(v T) { value = v }

	return get, set
}

func Computed[T any](fn func() T) func() T {
	return fn
}

func AsyncComputed[T any](fn func() (T, error)) func() (T, error) {
	return fn
}

func Batch(fn func()) {
	fn()
}

func Effect(fn func() func()) {
	fn()
}

func Untrack[T any](fn func() T) T {
	return fn()
}

func IsPending(fn func()) bool {
	return false
}

type context struct{ value any }

func Context(initial any) *context       { return &context{initial} }
func GetContext[T any](ctx *context) T   { return ctx.value.(T) }
func SetContext(ctx *context, value any) { ctx.value = value }

type owner struct{}

func Owner() *owner                     { return &owner{} }
func (o *owner) Run(fn func())          { fn() }
func (o *owner) Dispose()               {}
func (o *owner) OnError(fn func(error)) { fn(nil) }
func (o *owner) OnDispose(fn func())    { fn() }
