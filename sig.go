package sig

import "github.com/AnatoleLucet/sig/internal"

func readAs[T any](s *internal.Signal) func() T {
	return func() T { return s.Read().(T) }
}

func writeAs[T any](s *internal.Signal) func(T) {
	return func(v T) { s.Write(v) }
}

func Signal[T any](initial T) (func() T, func(T)) {
	s := internal.GetRuntime().NewSignal(initial)

	return readAs[T](s), writeAs[T](s)
}

func Computed[T any](compute func() T) func() T {
	c := internal.GetRuntime().NewComputed(func() any {
		return compute()
	})

	return readAs[T](c.Signal)
}

func AsyncComputed[T any](fn func() (T, error)) func() (T, error) {
	return fn
}

func Batch(fn func()) {
	fn()
}

func Effect(fn func() func()) {
	internal.GetRuntime().NewEffect(internal.EffectUser, fn)
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
