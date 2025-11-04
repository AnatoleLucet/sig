package sig

type EffectComputation interface {
	func() | func() func()
}

type effect[T EffectComputation] struct {
	*owner
	*dependencyTracker

	computation T
	cleanup     func()
}

func (e *effect[T]) addDependency(o Observable) {
	e.dependencyTracker.add(o)
}

func (e *effect[T]) removeDependency(o Observable) {
	e.dependencyTracker.remove(o)
}

func (e *effect[T]) clean() {
	if e.cleanup != nil {
		e.cleanup()
		e.cleanup = nil
	}

	e.dependencyTracker.clear(e)

	for _, child := range e.owner.children {
		child.Dispose()
	}
	e.owner.children = nil
}

func (e *effect[T]) Dispose() {
	e.clean()
	e.owner.Dispose()
}

func (e *effect[T]) Execute() {
	e.clean()

	prevReaction := e.parent.ctx.activeReaction
	e.parent.ctx.activeReaction = e

	e.Run(func() {
		switch fn := any(e.computation).(type) {
		case func():
			fn()
			e.cleanup = nil
		case func() func():
			e.cleanup = fn()
		}
	})

	e.parent.ctx.activeReaction = prevReaction
}

func Effect[T EffectComputation](computation T) *effect[T] {
	e := &effect[T]{
		owner:             &owner{parent: getActiveOwner(), ctx: &reactiveContext{}},
		dependencyTracker: &dependencyTracker{},

		computation: computation,
	}
	e.parent.addChild(e)

	e.Execute()

	return e
}
