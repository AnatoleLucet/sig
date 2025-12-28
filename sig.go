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
	c := internal.GetRuntime().NewComputed(func(node *internal.Computed) any {
		return compute()
	})

	return readAs[T](c.Signal)
}

func AsyncComputed[T any](fn func() (T, error)) func() (T, error) {
	return fn
}

func Batch(fn func()) {
	internal.GetRuntime().NewBatch(fn)
}

func Effect(fn func() func()) {
	internal.GetRuntime().NewEffect(internal.EffectUser, fn)
}

func Untrack[T any](fn func() T) T {
	var result T
	internal.GetRuntime().Untrack(func() { result = fn() })
	return result
}

func IsPending(fn func()) bool {
	return false
}

func OnCleanup(fn func()) {
	internal.GetRuntime().OnCleanup(fn)
}

type context struct{ value any }

func Context(initial any) *context       { return &context{initial} }
func GetContext[T any](ctx *context) T   { return ctx.value.(T) }
func SetContext(ctx *context, value any) { ctx.value = value }

type owner struct {
	owner *internal.Owner
}

func Owner() *owner {
	return &owner{internal.GetRuntime().NewOwner()}
}

func (o *owner) Run(fn func() error) error {
	return o.owner.Run(fn)
}
func (o *owner) Dispose() {
	o.owner.Dispose()
}
func (o *owner) OnCleanup(fn func()) {
	o.owner.OnCleanup(fn)
}
func (o *owner) OnDispose(fn func()) {
	o.owner.OnDispose(fn)
}
func (o *owner) OnError(fn func(any)) {
	o.owner.OnError(fn)
}
