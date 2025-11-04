package sig

type signal[T comparable] struct {
	*reactionTracker

	value T
	owner *owner
}

func (s *signal[T]) track(r Reaction) {
	s.reactionTracker.track(s, r)
}

func (s *signal[T]) untrack(r Reaction) {
	s.reactionTracker.untrack(s, r)
}

func (s *signal[T]) Get() T {
	if s.owner.ctx.activeReaction != nil {
		s.track(s.owner.ctx.activeReaction)
	}

	return s.value
}

func (s *signal[T]) Set(newValue T) {
	if s.value == newValue {
		return
	}

	s.value = newValue
	s.reactionTracker.react(s.owner.ctx)
}

func Signal[T comparable](initial T) (func() T, func(T)) {
	s := &signal[T]{
		reactionTracker: &reactionTracker{},

		value: initial,
		owner: getActiveOwner(),
	}

	return s.Get, s.Set
}
