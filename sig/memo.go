package sig

type memo[T any] struct {
	*reactionTracker
	*dependencyTracker

	dirty       bool
	computation func() T
	value       T

	owner *owner
}

func (m *memo[T]) addDependency(o Observable) {
	m.dependencyTracker.add(o)
}

func (m *memo[T]) removeDependency(o Observable) {
	m.dependencyTracker.remove(o)
}

func (m *memo[T]) track(r Reaction) {
	m.reactionTracker.track(m, r)
}

func (m *memo[T]) untrack(r Reaction) {
	m.reactionTracker.untrack(m, r)
}

func (m *memo[T]) Dispose() {
	m.dependencyTracker.clear(m)
	m.reactionTracker.clear(m)
}

func (m *memo[T]) Execute() {
	m.dirty = true
	m.reactionTracker.react(m.owner.ctx)
}

func (m *memo[T]) Get() T {
	if m.owner.ctx.activeReaction != nil {
		m.track(m.owner.ctx.activeReaction)
	}

	if !m.dirty {
		return m.value
	}

	// clear previous dependencies
	m.dependencyTracker.clear(m)

	prevReaction := m.owner.ctx.activeReaction
	m.owner.ctx.activeReaction = m

	m.value = m.computation()
	m.dirty = false

	m.owner.ctx.activeReaction = prevReaction

	return m.value
}

func Memo[T any](computation func() T) func() T {
	m := &memo[T]{
		reactionTracker:   &reactionTracker{},
		dependencyTracker: &dependencyTracker{},

		dirty:       true,
		computation: computation,

		owner: getActiveOwner(),
	}
	m.owner.addChild(m)

	return m.Get
}
